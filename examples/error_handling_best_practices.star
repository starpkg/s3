#!/usr/bin/env starcli
"""
Error Handling Best Practices Example

This example demonstrates robust error handling patterns for S3 operations:
- Input validation and sanitization
- Graceful error handling with detailed messages
- Retry logic with exponential backoff
- Fallback strategies and recovery
- Error logging and reporting

Usage: starcli error_handling_best_practices.star [bucket-name]
"""

load("s3", "create_client", "validate_bucket_name", "validate_object_key")
load("time")

def main():
    """Demonstrate error handling best practices"""
    
    args = runtime.args[1:]
    bucket_name = args[0] if len(args) > 0 else "error-handling-demo-bucket"
    
    print("=== S3 Error Handling Best Practices Demo ===")
    print("Bucket: {}".format(bucket_name))
    print()
    
    # Create S3 client
    s3 = create_client(aws_region="us-east-1")
    
    # Demo 1: Input validation
    demonstrate_input_validation()
    
    # Demo 2: Safe operations with error handling
    demonstrate_safe_operations(s3, bucket_name)
    
    # Demo 3: Retry logic
    demonstrate_retry_logic(s3, bucket_name)
    
    # Demo 4: Graceful degradation
    demonstrate_graceful_degradation(s3, bucket_name)
    
    print("\n=== Error handling demo completed! ===")

def demonstrate_input_validation():
    """Show input validation best practices"""
    print("1. Input Validation Examples:")
    
    # Test bucket name validation
    test_bucket_names = [
        "valid-bucket-name",
        "INVALID-BUCKET-NAME",  # Contains uppercase
        "bucket_with_underscores",  # Contains underscores
        "a",  # Too short
        "my-bucket-name-is-way-too-long-and-exceeds-the-maximum-length-limit",  # Too long
        "bucket..name",  # Double periods
        "bucket-",  # Ends with hyphen
        "192.168.1.1"  # IP address format
    ]
    
    for bucket_name in test_bucket_names:
        is_valid = validate_bucket_name(bucket_name)
        status = "✓ Valid" if is_valid else "❌ Invalid"
        print("   {} - {}".format(bucket_name, status))
    
    print()
    
    # Test object key validation
    test_object_keys = [
        "valid/object/key.txt",
        "file with spaces.txt",
        "unicode-文件名.txt",
        "very/deep/nested/directory/structure/file.txt",
        "",  # Empty key
        "/leading-slash.txt",  # Leading slash
        "trailing-slash/",  # Trailing slash
        "normal-file.txt"
    ]
    
    print("   Object Key Validation:")
    for object_key in test_object_keys:
        is_valid = validate_object_key(object_key)
        status = "✓ Valid" if is_valid else "❌ Invalid"
        display_key = object_key if object_key else "(empty)"
        print("   {} - {}".format(display_key, status))
    
    print()

def demonstrate_safe_operations(s3, bucket_name):
    """Show safe operation patterns with comprehensive error handling"""
    print("2. Safe Operations with Error Handling:")
    
    # Safe bucket creation
    safe_create_bucket(s3, bucket_name)
    
    # Safe file upload with validation
    safe_upload_file(s3, bucket_name, "test/safe-upload.txt", "Safe upload content")
    
    # Safe file download with fallback
    content = safe_download_file(s3, bucket_name, "test/safe-upload.txt", "Default content")
    print("   Downloaded content: {}".format(content))
    
    # Safe file deletion
    safe_delete_file(s3, bucket_name, "test/safe-upload.txt")
    
    print()

def safe_create_bucket(s3, bucket_name):
    """Safely create a bucket with comprehensive error handling"""
    print("   Creating bucket safely...")
    
    # Validate bucket name first
    if not validate_bucket_name(bucket_name):
        print("     ❌ Invalid bucket name: {}".format(bucket_name))
        return False
    
    try:
        # Check if bucket already exists
        if s3.bucket_exists(bucket_name):
            print("     ✓ Bucket '{}' already exists".format(bucket_name))
            return True
        
        # Create the bucket
        s3.create_bucket(bucket_name)
        print("     ✓ Bucket '{}' created successfully".format(bucket_name))
        return True
        
    except Exception as e:
        error_msg = str(e).lower()
        
        # Provide specific error messages based on error type
        if "already exists" in error_msg:
            print("     ⚠️  Bucket already exists (created by another process)")
            return True
        elif "access denied" in error_msg:
            print("     ❌ Access denied - check credentials and permissions")
            return False
        elif "invalid" in error_msg:
            print("     ❌ Invalid bucket configuration: {}".format(e))
            return False
        else:
            print("     ❌ Unexpected error creating bucket: {}".format(e))
            return False

def safe_upload_file(s3, bucket_name, object_key, content):
    """Safely upload a file with validation and error handling"""
    print("   Uploading file safely...")
    
    # Validate inputs
    if not validate_object_key(object_key):
        print("     ❌ Invalid object key: {}".format(object_key))
        return False
    
    if content == None or content == "":
        print("     ❌ Content cannot be empty")
        return False
    
    try:
        # Ensure bucket exists
        if not s3.bucket_exists(bucket_name):
            print("     ⚠️  Bucket doesn't exist, creating it...")
            if not safe_create_bucket(s3, bucket_name):
                return False
        
        # Upload the file
        s3.put_object(
            bucket_name,
            object_key,
            content,
            metadata={
                "uploaded-by": "error-handling-demo",
                "upload-time": time.now().format("2006-01-02T15:04:05Z")
            }
        )
        
        print("     ✓ File uploaded successfully: s3://{}/{}".format(bucket_name, object_key))
        return True
        
    except Exception as e:
        error_msg = str(e).lower()
        
        if "access denied" in error_msg:
            print("     ❌ Upload failed - access denied")
        elif "bucket" in error_msg and "not" in error_msg:
            print("     ❌ Upload failed - bucket issue: {}".format(e))
        elif "size" in error_msg or "large" in error_msg:
            print("     ❌ Upload failed - file too large")
        else:
            print("     ❌ Upload failed - unexpected error: {}".format(e))
        
        return False

def safe_download_file(s3, bucket_name, object_key, fallback_content=None):
    """Safely download a file with fallback options"""
    print("   Downloading file safely...")
    
    try:
        # Check if object exists first
        if not s3.object_exists(bucket_name, object_key):
            print("     ⚠️  Object doesn't exist: s3://{}/{}".format(bucket_name, object_key))
            if fallback_content != None:
                print("     ✓ Using fallback content")
                return fallback_content
            else:
                return None
        
        # Get object info to check size
        info = s3.get_object_info(bucket_name, object_key)
        size_mb = info["size"] / (1024 * 1024)
        
        # Check size limit (100MB for this demo)
        if size_mb > 100:
            print("     ❌ File too large: {:.2f}MB (max 100MB)".format(size_mb))
            return fallback_content
        
        # Download the file
        content = s3.get_object(bucket_name, object_key)
        print("     ✓ File downloaded successfully ({} bytes)".format(len(content)))
        return content
        
    except Exception as e:
        error_msg = str(e).lower()
        
        if "not found" in error_msg or "no such key" in error_msg:
            print("     ⚠️  File not found, using fallback")
            return fallback_content
        elif "access denied" in error_msg:
            print("     ❌ Download failed - access denied")
            return fallback_content
        elif "timeout" in error_msg:
            print("     ❌ Download failed - timeout")
            return fallback_content
        else:
            print("     ❌ Download failed - unexpected error: {}".format(e))
            return fallback_content

def safe_delete_file(s3, bucket_name, object_key):
    """Safely delete a file with error handling"""
    print("   Deleting file safely...")
    
    try:
        # Check if object exists
        if not s3.object_exists(bucket_name, object_key):
            print("     ✓ Object doesn't exist (already deleted)")
            return True
        
        # Delete the object
        s3.delete_object(bucket_name, object_key)
        
        # Verify deletion
        if not s3.object_exists(bucket_name, object_key):
            print("     ✓ File deleted successfully")
            return True
        else:
            print("     ⚠️  File still exists after deletion attempt")
            return False
            
    except Exception as e:
        error_msg = str(e).lower()
        
        if "not found" in error_msg:
            print("     ✓ File already deleted")
            return True
        elif "access denied" in error_msg:
            print("     ❌ Delete failed - access denied")
            return False
        else:
            print("     ❌ Delete failed - unexpected error: {}".format(e))
            return False

def demonstrate_retry_logic(s3, bucket_name):
    """Show retry logic with exponential backoff"""
    print("3. Retry Logic with Exponential Backoff:")
    
    # Demonstrate retry for upload operation
    retry_upload_with_backoff(s3, bucket_name, "test/retry-demo.txt", "Retry demo content")
    
    # Demonstrate retry for download operation
    retry_download_with_backoff(s3, bucket_name, "test/retry-demo.txt")
    
    print()

def retry_upload_with_backoff(s3, bucket_name, object_key, content, max_attempts=3):
    """Upload with retry logic and exponential backoff"""
    print("   Upload with retry logic...")
    
    for attempt in range(1, max_attempts + 1):
        try:
            s3.put_object(bucket_name, object_key, content)
            print("     ✓ Upload successful on attempt {}".format(attempt))
            return True
            
        except Exception as e:
            error_msg = str(e).lower()
            
            # Determine if error is retryable
            retryable_errors = [
                "timeout",
                "service unavailable",
                "internal server error",
                "throttling",
                "slow down",
                "connection",
                "network"
            ]
            
            is_retryable = any(retry_error in error_msg for retry_error in retryable_errors)
            
            if not is_retryable or attempt == max_attempts:
                print("     ❌ Upload failed after {} attempts: {}".format(attempt, e))
                return False
            
            # Calculate backoff delay (exponential backoff with jitter)
            base_delay = 2 ** (attempt - 1)  # 1, 2, 4, 8, 16...
            max_delay = min(base_delay, 30)  # Cap at 30 seconds
            
            print("     ⚠️  Attempt {} failed ({}), retrying in {} seconds...".format(
                attempt, e, max_delay))
            
            # Sleep for backoff period
            time.sleep(max_delay)
    
    return False

def retry_download_with_backoff(s3, bucket_name, object_key, max_attempts=3):
    """Download with retry logic"""
    print("   Download with retry logic...")
    
    for attempt in range(1, max_attempts + 1):
        try:
            content = s3.get_object(bucket_name, object_key)
            print("     ✓ Download successful on attempt {} ({} bytes)".format(
                attempt, len(content)))
            return content
            
        except Exception as e:
            error_msg = str(e).lower()
            
            # Non-retryable errors
            if "not found" in error_msg or "no such key" in error_msg:
                print("     ❌ Object not found: {}".format(object_key))
                return None
            
            # Retryable errors
            retryable_errors = ["timeout", "connection", "network", "service"]
            is_retryable = any(retry_error in error_msg for retry_error in retryable_errors)
            
            if not is_retryable or attempt == max_attempts:
                print("     ❌ Download failed after {} attempts: {}".format(attempt, e))
                return None
            
            # Exponential backoff
            delay = min(2 ** (attempt - 1), 30)
            print("     ⚠️  Attempt {} failed, retrying in {} seconds...".format(attempt, delay))
            time.sleep(delay)
    
    return None

def demonstrate_graceful_degradation(s3, bucket_name):
    """Show graceful degradation patterns"""
    print("4. Graceful Degradation Examples:")
    
    # Feature detection and fallback
    demonstrate_feature_detection(s3, bucket_name)
    
    # Batch operations with partial success handling
    demonstrate_batch_operations(s3, bucket_name)
    
    print()

def demonstrate_feature_detection(s3, bucket_name):
    """Demonstrate feature detection and graceful fallback"""
    print("   Feature detection and fallback...")
    
    # Test versioning support
    try:
        versioning = s3.get_bucket_versioning(bucket_name)
        print("     ✓ Versioning supported - status: {}".format(
            "enabled" if versioning.get("enabled") else "disabled"))
    except Exception:
        print("     ⚠️  Versioning not supported by this service")
    
    # Test tagging support
    test_key = "test/feature-detection.txt"
    s3.put_object(bucket_name, test_key, "Feature detection test")
    
    try:
        tags = {"test": "feature-detection", "service": "s3"}
        s3.set_object_tags(bucket_name, test_key, tags)
        retrieved_tags = s3.get_object_tags(bucket_name, test_key)
        if len(retrieved_tags) > 0:
            print("     ✓ Object tagging supported")
        else:
            print("     ⚠️  Object tagging not fully supported")
    except Exception:
        print("     ⚠️  Object tagging not supported by this service")
    
    # Test pre-signed URLs
    try:
        url = s3.presign_url(bucket_name, test_key, expires_in=3600)
        if url and url.startswith("http"):
            print("     ✓ Pre-signed URLs supported")
        else:
            print("     ⚠️  Pre-signed URLs not supported")
    except Exception:
        print("     ⚠️  Pre-signed URLs not supported by this service")
    
    # Cleanup
    s3.delete_object(bucket_name, test_key)

def demonstrate_batch_operations(s3, bucket_name):
    """Demonstrate batch operations with partial success handling"""
    print("   Batch operations with error handling...")
    
    # Create multiple test files
    test_files = {
        "batch/file1.txt": "Content for file 1",
        "batch/file2.txt": "Content for file 2", 
        "batch/file3.txt": "Content for file 3",
        "batch/file4.txt": "Content for file 4",
        "batch/file5.txt": "Content for file 5"
    }
    
    # Upload files individually with error tracking
    successful_uploads = []
    failed_uploads = []
    
    for file_key, content in test_files.items():
        try:
            s3.put_object(bucket_name, file_key, content)
            successful_uploads.append(file_key)
        except Exception as e:
            failed_uploads.append({"key": file_key, "error": str(e)})
    
    print("     Batch upload results:")
    print("       ✓ Successful: {} files".format(len(successful_uploads)))
    print("       ❌ Failed: {} files".format(len(failed_uploads)))
    
    # Batch delete with error handling
    if len(successful_uploads) > 0:
        try:
            delete_result = s3.delete_objects(bucket_name, successful_uploads)
            deleted_count = len(delete_result.get("deleted", []))
            errors = delete_result.get("errors", [])
            
            print("     Batch delete results:")
            print("       ✓ Deleted: {} files".format(deleted_count))
            
            if len(errors) > 0:
                print("       ❌ Delete errors: {} files".format(len(errors)))
                for error in errors:
                    print("         {}: {}".format(error.get("key"), error.get("message")))
                    
        except Exception as e:
            print("     ❌ Batch delete failed: {}".format(e))
            
            # Fallback to individual deletions
            print("     🔄 Falling back to individual deletions...")
            for file_key in successful_uploads:
                try:
                    s3.delete_object(bucket_name, file_key)
                    print("       ✓ Deleted: {}".format(file_key))
                except Exception as delete_error:
                    print("       ❌ Failed to delete {}: {}".format(file_key, delete_error))

# Run the demo
main() 