#!/usr/bin/env starcli
"""
Basic File Management Example

This example demonstrates fundamental S3 operations including:
- Creating and checking buckets
- Uploading and downloading files
- Listing objects
- Setting metadata and content types
- Generating pre-signed URLs
- Basic error handling

Usage: starcli basic_file_management.star [bucket-name]
"""

load("s3", "create_client")

def main():
    """Main function demonstrating basic S3 operations"""
    
    # Get bucket name from command line or use default
    args = runtime.args[1:]
    bucket_name = args[0] if len(args) > 0 else "my-basic-files-bucket"
    
    print("=== Basic S3 File Management Demo ===")
    print("Using bucket:", bucket_name)
    print()
    
    # Create S3 client (uses environment variables for credentials)
    s3 = create_client(region="us-east-1")
    
    # Step 1: Ensure bucket exists
    ensure_bucket_exists(s3, bucket_name)
    
    # Step 2: Upload various types of files
    upload_sample_files(s3, bucket_name)
    
    # Step 3: List and inspect objects
    list_bucket_contents(s3, bucket_name)
    
    # Step 4: Download files
    download_sample_files(s3, bucket_name)
    
    # Step 5: Generate pre-signed URLs
    generate_download_links(s3, bucket_name)
    
    # Step 6: Cleanup (optional)
    # cleanup_demo_files(s3, bucket_name)
    
    print("\n=== Demo completed successfully! ===")

def ensure_bucket_exists(s3, bucket_name):
    """Ensure the bucket exists, create if it doesn't"""
    print("1. Checking bucket existence...")
    
    if s3.bucket_exists(bucket_name):
        print("✓ Bucket '{}' already exists".format(bucket_name))
        
        # Get bucket location
        location = s3.get_bucket_location(bucket_name)
        print("  Region: {}".format(location))
    else:
        print("✓ Creating bucket '{}'...".format(bucket_name))
        s3.create_bucket(bucket_name)
        print("  Bucket created successfully")
    
    print()

def upload_sample_files(s3, bucket_name):
    """Upload various types of sample files"""
    print("2. Uploading sample files...")
    
    # Upload a simple text file
    s3.put_object(
        bucket_name, 
        "hello.txt", 
        "Hello from Starlark S3 module!\nThis is a basic text file.",
        content_type="text/plain"
    )
    print("✓ Uploaded: hello.txt")
    
    # Upload JSON data with metadata
    json_data = '{"message": "Hello World", "timestamp": "2024-01-15T10:30:00Z", "version": "1.0"}'
    s3.put_object(
        bucket_name,
        "data/config.json",
        json_data,
        content_type="application/json",
        metadata={
            "created-by": "starlark-demo",
            "file-type": "configuration",
            "version": "1.0"
        }
    )
    print("✓ Uploaded: data/config.json (with metadata)")
    
    # Upload HTML content with cache headers
    html_content = """
<!DOCTYPE html>
<html>
<head>
    <title>S3 Demo Page</title>
</head>
<body>
    <h1>Hello from S3!</h1>
    <p>This page was uploaded using the Starlark S3 module.</p>
</body>
</html>
"""
    s3.put_object(
        bucket_name,
        "web/index.html",
        html_content,
        content_type="text/html",
        metadata={
            "cache-control": "public, max-age=3600"
        }
    )
    print("✓ Uploaded: web/index.html (with cache control)")
    
    # Upload with tags
    s3.put_object(
        bucket_name,
        "logs/app.log",
        "2024-01-15 10:30:00 INFO Application started\n2024-01-15 10:30:01 INFO Ready to serve requests\n",
        content_type="text/plain",
        tags={
            "environment": "demo",
            "application": "s3-example",
            "log-level": "info"
        }
    )
    print("✓ Uploaded: logs/app.log (with tags)")
    
    print()

def list_bucket_contents(s3, bucket_name):
    """List and inspect bucket contents"""
    print("3. Listing bucket contents...")
    
    # List all objects
    result = s3.list_objects(bucket_name)
    
    if len(result["contents"]) == 0:
        print("  No objects found in bucket")
        return
    
    print("  Found {} objects:".format(len(result["contents"])))
    
    for obj in result["contents"]:
        # Get detailed object information
        info = s3.get_object_info(bucket_name, obj["key"])
        
        print("    📄 {}".format(obj["key"]))
        print("       Size: {} bytes".format(obj["size"]))
        print("       Modified: {}".format(obj["last_modified"]))
        print("       Content-Type: {}".format(info.get("content_type", "unknown")))
        
        # Show metadata if present
        metadata = info.get("metadata", {})
        if len(metadata) > 0:
            print("       Metadata:")
            for key, value in metadata.items():
                print("         {}: {}".format(key, value))
        
        # Show tags if present (try to get them, may not be supported by all services)
        try:
            tags = s3.get_object_tags(bucket_name, obj["key"])
            if len(tags) > 0:
                print("       Tags:")
                for key, value in tags.items():
                    print("         {}: {}".format(key, value))
        except Exception:
            # Tags may not be supported by all S3-compatible services
            pass
        
        print()

def download_sample_files(s3, bucket_name):
    """Download and verify sample files"""
    print("4. Downloading sample files...")
    
    # Download and display text file
    try:
        content = s3.get_object(bucket_name, "hello.txt")
        print("✓ Downloaded hello.txt:")
        print("  Content: {}".format(content.replace("\n", "\\n")))
    except Exception as e:
        print("✗ Failed to download hello.txt: {}".format(e))
    
    # Download and parse JSON
    try:
        json_content = s3.get_object(bucket_name, "data/config.json")
        print("✓ Downloaded data/config.json:")
        print("  Content: {}".format(json_content))
        
        # Parse JSON if json module is available
        try:
            load("json", "decode")
            data = decode(json_content)
            print("  Parsed message: {}".format(data.get("message", "N/A")))
        except Exception:
            print("  (JSON parsing not available)")
    except Exception as e:
        print("✗ Failed to download data/config.json: {}".format(e))
    
    print()

def generate_download_links(s3, bucket_name):
    """Generate pre-signed URLs for downloading"""
    print("5. Generating pre-signed download URLs...")
    
    files_to_share = ["hello.txt", "data/config.json", "web/index.html"]
    
    for file_key in files_to_share:
        if s3.object_exists(bucket_name, file_key):
            try:
                # Generate 1-hour download link
                url = s3.presign_url(bucket_name, file_key, expires_in=3600)
                print("✓ Download link for '{}':".format(file_key))
                print("  {}".format(url))
                print("  (Valid for 1 hour)")
            except Exception as e:
                print("✗ Failed to generate URL for '{}': {}".format(file_key, e))
        else:
            print("✗ File '{}' does not exist".format(file_key))
    
    print()

def cleanup_demo_files(s3, bucket_name):
    """Clean up demo files (optional)"""
    print("6. Cleaning up demo files...")
    
    # List all demo files
    result = s3.list_objects(bucket_name)
    demo_files = [obj["key"] for obj in result["contents"]]
    
    if len(demo_files) > 0:
        print("  Deleting {} files...".format(len(demo_files)))
        delete_result = s3.delete_objects(bucket_name, demo_files)
        
        deleted_count = len(delete_result.get("deleted", []))
        errors = delete_result.get("errors", [])
        
        print("  ✓ Deleted {} files".format(deleted_count))
        
        if len(errors) > 0:
            print("  ✗ Errors occurred:")
            for error in errors:
                print("    {}: {}".format(error["key"], error["message"]))
    else:
        print("  No files to clean up")
    
    print()

# Run the demo
main() 