# S3 Starlark Module Development Plan

## 🗂️ S3 Module - Simple Storage Service Operations for Starlark

**Module Name**: `s3`  
**Emoji**: 🗂️  
**Description**: Complete S3-compatible storage operations for Starlark  
**Tagline**: Unified interface for Amazon S3, MinIO, and all S3-compatible storage services

## Executive Summary

The `s3` module provides comprehensive S3-compatible storage operations for Starlark scripts. It focuses on simplicity, security, and performance while supporting all major S3-compatible services including Amazon S3, MinIO, DigitalOcean Spaces, Backblaze B2, and more. The design emphasizes ease of use with powerful features for both simple scripts and complex applications.

## Core Design Principles

1. **Function-based API**: Uses `client()` function instead of class constructors
2. **S3-compatible First**: Works seamlessly with any S3-compatible service
3. **Security by Default**: Secure credential handling with base package integration
4. **High Performance**: Optimized for large files with streaming and concurrent operations
5. **Starlark Native**: Designed specifically for Starlark constraints and patterns
6. **Production Ready**: Built for reliability with proper error handling and retries

## Starlark Constraints & Adaptations

### Key Limitations Addressed

- ❌ **No Classes**: Use `client()` function returning object with methods
- ❌ **No f-strings**: Use `.format()` method for string formatting
- ❌ **No try/except**: Use `fail()` for error handling and None checks
- ❌ **No `is`/`is not`**: Use `== None` and `!= None`
- ❌ **No while loops**: Use for loops with range when needed
- ❌ **Limited imports**: Function-based module loading with `load()`

## Core Features

### 1. **Client Management**

- Multiple authentication methods (keys, environment, IAM roles)
- Support for all S3-compatible services
- Configurable endpoints, regions, and credentials
- SSL/TLS configuration options
- Request timeout and retry configuration
- Connection pooling and reuse

### 2. **Bucket Operations**

- Create buckets with region selection
- Delete empty and non-empty buckets
- List all buckets with metadata
- Check bucket existence
- Get bucket location/region
- Configure bucket versioning
- Manage bucket policies
- Lifecycle rule management

### 3. **Object Operations**

- Upload objects (strings, bytes, files)
- Download objects (to memory or file)
- Delete single or multiple objects
- List objects with prefix/delimiter support
- Copy objects within or across buckets
- Move objects (copy + delete)
- Get object metadata and properties
- Set custom metadata and tags
- Generate pre-signed URLs

### 4. **Advanced Features**

- Multi-part upload for large files
- Stream-based uploads/downloads
- Batch operations for efficiency
- Server-side encryption options
- Object versioning support
- Progress tracking capabilities

## API Design

### Core Module Functions

```python
# Client creation
client(access_key_id=None, secret_access_key=None, region="us-east-1", **config) -> S3Client

# Response builders (for advanced use cases)
upload_options(content_type=None, metadata={}, tags={}) -> Options
multipart_options(part_size=5*1024*1024, parallel_uploads=3) -> Options

# Utility functions
parse_s3_url(url) -> {"bucket": str, "key": str}
generate_s3_url(bucket, key, region="us-east-1") -> str
validate_bucket_name(name) -> bool
validate_object_key(key) -> bool
```

### Client Creation Examples

```python
load("s3", "client")

# Create a client with AWS credentials
s3 = client(
    access_key_id="YOUR_ACCESS_KEY",
    secret_access_key="YOUR_SECRET_KEY",
    region="us-east-1"
)

# Or use environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
s3 = client()

# For S3-compatible services (e.g., MinIO)
s3 = client(
    endpoint="http://localhost:9000",
    access_key_id="minioadmin",
    secret_access_key="minioadmin",
    region="us-east-1",
    force_path_style=True,  # Required for MinIO
    use_ssl=False
)

# With advanced configuration
s3 = client(
    region="eu-west-1",
    timeout=60,
    max_retries=5,
    enable_compression=True
)
```

### S3Client Object API

```python
# Bucket operations
s3.create_bucket(name, region=None, **options)
s3.delete_bucket(name, force=False)
s3.list_buckets() -> list
s3.bucket_exists(name) -> bool
s3.get_bucket_location(name) -> str
s3.set_bucket_versioning(name, enabled=True)
s3.get_bucket_versioning(name) -> dict

# Object operations - core
s3.put_object(bucket, key, content, **options)
s3.put_object_from_file(bucket, key, file_path, **options)
s3.get_object(bucket, key) -> str
s3.get_object_to_file(bucket, key, file_path)
s3.delete_object(bucket, key)
s3.delete_objects(bucket, keys) -> dict

# Object operations - advanced
s3.copy_object(src_bucket, src_key, dst_bucket, dst_key, **options)
s3.move_object(src_bucket, src_key, dst_bucket, dst_key, **options)
s3.list_objects(bucket, prefix="", delimiter="", max_keys=1000) -> dict
s3.get_object_info(bucket, key) -> dict
s3.object_exists(bucket, key) -> bool

# Metadata and tagging
s3.get_object_metadata(bucket, key) -> dict
s3.set_object_metadata(bucket, key, metadata)
s3.get_object_tags(bucket, key) -> dict
s3.set_object_tags(bucket, key, tags)
s3.delete_object_tags(bucket, key)

# Pre-signed URLs
s3.presign_url(bucket, key, expires_in=3600, method="GET") -> str
s3.presign_put_url(bucket, key, expires_in=3600, **options) -> str
s3.presign_post(bucket, key, expires_in=3600, **options) -> dict

# Multi-part upload
s3.create_multipart_upload(bucket, key, **options) -> str
s3.upload_part(bucket, key, upload_id, part_number, content) -> dict
s3.complete_multipart_upload(bucket, key, upload_id, parts) -> dict
s3.abort_multipart_upload(bucket, key, upload_id)
s3.list_multipart_uploads(bucket, prefix="") -> list
```

### Bucket Operations Examples

```python
# Create a bucket
s3.create_bucket("my-bucket")
s3.create_bucket("my-bucket", region="eu-west-1")

# List buckets
buckets = s3.list_buckets()
for bucket in buckets:
    print("Bucket: {}, Created: {}".format(bucket["name"], bucket["creation_date"]))

# Check if bucket exists
if s3.bucket_exists("my-bucket"):
    print("Bucket exists!")
else:
    s3.create_bucket("my-bucket")

# Delete bucket
s3.delete_bucket("old-bucket")
s3.delete_bucket("full-bucket", force=True)  # Delete non-empty bucket

# Get bucket location
location = s3.get_bucket_location("my-bucket")
print("Bucket region:", location)

# Configure versioning
s3.set_bucket_versioning("my-bucket", enabled=True)
versioning = s3.get_bucket_versioning("my-bucket")
print("Versioning enabled:", versioning["enabled"])
```

### Object Operations Examples

```python
# Upload objects
s3.put_object("my-bucket", "hello.txt", "Hello, World!")
s3.put_object("my-bucket", "data.json", '{"key": "value"}', content_type="application/json")
s3.put_object_from_file("my-bucket", "image.jpg", "/path/to/image.jpg")

# Download objects
content = s3.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"

s3.get_object_to_file("my-bucket", "image.jpg", "/local/path/image.jpg")

# List objects
objects = s3.list_objects("my-bucket")
for obj in objects["contents"]:
    print("{} ({} bytes)".format(obj["key"], obj["size"]))

# List with prefix and delimiter
photos = s3.list_objects("my-bucket", prefix="photos/2024/", delimiter="/")
for obj in photos["contents"]:
    print(obj["key"])

# Delete objects
s3.delete_object("my-bucket", "hello.txt")
result = s3.delete_objects("my-bucket", ["file1.txt", "file2.txt", "file3.txt"])
print("Deleted:", len(result["deleted"]))

# Copy objects
s3.copy_object("source-bucket", "source.txt", "dest-bucket", "dest.txt")

# Get object info
info = s3.get_object_info("my-bucket", "hello.txt")
print("Size: {} bytes".format(info["size"]))
print("Last modified: {}".format(info["last_modified"]))
print("ETag: {}".format(info["etag"]))

# Check if object exists
if s3.object_exists("my-bucket", "hello.txt"):
    print("Object exists!")

# Generate pre-signed URL
url = s3.presign_url("my-bucket", "private.pdf", expires_in=3600)
print("Download URL:", url)
```

### Metadata and Tags Examples

```python
# Upload with custom metadata
s3.put_object(
    "my-bucket",
    "document.pdf",
    pdf_content,
    metadata={
        "author": "Jane Doe",
        "department": "Engineering",
        "version": "2.1"
    }
)

# Retrieve metadata
metadata = s3.get_object_metadata("my-bucket", "document.pdf")
print("Author:", metadata.get("author"))
print("Version:", metadata.get("version"))

# Set object tags
s3.set_object_tags(
    "my-bucket", 
    "report.pdf",
    {
        "environment": "production",
        "confidential": "true",
        "project": "alpha"
    }
)

# Get object tags
tags = s3.get_object_tags("my-bucket", "report.pdf")
for key, value in tags.items():
    print("{}: {}".format(key, value))

# Delete object tags
s3.delete_object_tags("my-bucket", "report.pdf")
```

### Multi-part Upload Examples

```python
# Initiate multi-part upload
upload_id = s3.create_multipart_upload(
    "backup-bucket",
    "large-backup.tar.gz",
    content_type="application/gzip"
)

# Upload parts (example with file reading)
parts = []
part_size = 5 * 1024 * 1024  # 5MB chunks
part_number = 1

# Read file content (in practice, you'd read from file)
file_content = "large file content here..."  # This would be actual file data
total_size = len(file_content)

# Split content into parts
for offset in range(0, total_size, part_size):
    part_data = file_content[offset:offset + part_size]
    
    part = s3.upload_part(
        "backup-bucket",
        "large-backup.tar.gz",
        upload_id,
        part_number,
        part_data
    )
    parts.append(part)
    part_number = part_number + 1

# Complete the upload
result = s3.complete_multipart_upload(
    "backup-bucket",
    "large-backup.tar.gz",
    upload_id,
    parts
)
print("Upload completed. ETag:", result["etag"])

# Alternative: abort if something goes wrong
# s3.abort_multipart_upload("backup-bucket", "large-backup.tar.gz", upload_id)

# List ongoing multipart uploads
uploads = s3.list_multipart_uploads("backup-bucket")
for upload in uploads:
    print("Upload ID: {}, Key: {}".format(upload["upload_id"], upload["key"]))
```

## Complete Usage Examples

### 1. Basic File Management

```python
load("s3", "client")

def main():
    s3 = client(region="us-east-1")
    
    bucket_name = "my-files-bucket"
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating bucket:", bucket_name)
        s3.create_bucket(bucket_name)
    
    # Upload a simple text file
    s3.put_object(bucket_name, "hello.txt", "Hello from Starlark!")
    
    # Upload with metadata
    s3.put_object(
        bucket_name,
        "report.pdf",
        "PDF content here...",
        content_type="application/pdf",
        metadata={
            "author": "John Doe",
            "created": "2024-01-15"
        }
    )
    
    # List all objects
    objects = s3.list_objects(bucket_name)
    print("Objects in bucket:")
    for obj in objects["contents"]:
        print("  {} ({} bytes)".format(obj["key"], obj["size"]))
    
    # Download and print content
    content = s3.get_object(bucket_name, "hello.txt")
    print("Downloaded content:", content)
    
    # Generate a download link
    url = s3.presign_url(bucket_name, "report.pdf", expires_in=3600)
    print("Download URL:", url)

main()
```

### 2. Website Static File Deployment

```python
load("s3", "client")
load("file", "exists", "read")
load("path", "join", "ext")

def deploy_website(bucket_name, local_dir):
    """Deploy a static website to S3"""
    
    s3 = client()
    
    # Content type mapping
    content_types = {
        ".html": "text/html",
        ".css": "text/css",
        ".js": "application/javascript",
        ".json": "application/json",
        ".png": "image/png",
        ".jpg": "image/jpeg",
        ".jpeg": "image/jpeg",
        ".gif": "image/gif",
        ".svg": "image/svg+xml",
        ".ico": "image/x-icon",
        ".woff": "font/woff",
        ".woff2": "font/woff2"
    }
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating bucket:", bucket_name)
        s3.create_bucket(bucket_name)
    
    # Files to upload (simplified - in real usage you'd scan directory)
    files_to_upload = [
        "index.html",
        "about.html",
        "css/style.css",
        "js/app.js",
        "images/logo.png"
    ]
    
    uploaded_count = 0
    
    for file_path in files_to_upload:
        local_path = join(local_dir, file_path)
        
        if not exists(local_path):
            print("File not found:", local_path)
            continue
        
        # Determine content type
        file_ext = ext(file_path)
        content_type = content_types.get(file_ext, "application/octet-stream")
        
        # Set cache control for static assets
        cache_control = "public, max-age=3600"
        if file_ext in [".css", ".js", ".png", ".jpg", ".jpeg", ".gif"]:
            cache_control = "public, max-age=86400"  # 24 hours
        
        print("Uploading: {} -> s3://{}/{}".format(local_path, bucket_name, file_path))
        
        s3.put_object_from_file(
            bucket_name,
            file_path,
            local_path,
            content_type=content_type,
            metadata={"cache-control": cache_control}
        )
        uploaded_count = uploaded_count + 1
    
    print("Successfully uploaded {} files".format(uploaded_count))
    
    # Generate website URL (if using S3 website hosting)
    website_url = "http://{}.s3-website-{}.amazonaws.com".format(
        bucket_name, 
        s3.get_bucket_location(bucket_name) or "us-east-1"
    )
    print("Website URL:", website_url)

def main():
    deploy_website("my-website-bucket", "./dist")

main()
```

### 3. Backup System

```python
load("s3", "client")
load("time")
load("file", "read", "exists")
load("path", "join")

def backup_files(bucket_name, files_to_backup):
    """Backup files to S3 with timestamp and metadata"""
    
    s3 = client()
    timestamp = time.now().format("2006-01-02-15-04-05")
    
    # Ensure backup bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating backup bucket:", bucket_name)
        s3.create_bucket(bucket_name)
    
    # Backup each file
    for local_file in files_to_backup:
        if not exists(local_file):
            print("File not found, skipping:", local_file)
            continue
        
        # Create backup key with timestamp
        backup_key = "backups/{}/{}".format(timestamp, local_file.replace("/", "_"))
        
        print("Backing up: {} -> s3://{}/{}".format(local_file, bucket_name, backup_key))
        
        # Upload with backup metadata
        s3.put_object_from_file(
            bucket_name,
            backup_key,
            local_file,
            metadata={
                "backup-date": time.now().format(time.RFC3339),
                "original-path": local_file,
                "backup-type": "manual"
            },
            tags={
                "backup": "true",
                "date": timestamp,
                "retention": "30days"
            }
        )
    
    print("Backup completed at:", timestamp)
    return timestamp

def list_backups(bucket_name, days=30):
    """List recent backups"""
    
    s3 = client()
    
    # List backup objects
    result = s3.list_objects(bucket_name, prefix="backups/", max_keys=1000)
    
    print("Recent backups:")
    for obj in result["contents"]:
        # Get object metadata to show backup info
        try:
            info = s3.get_object_info(bucket_name, obj["key"])
            metadata = info.get("metadata", {})
            
            print("  {} ({}MB) - Original: {}".format(
                obj["key"],
                round(obj["size"] / (1024*1024), 2),
                metadata.get("original-path", "unknown")
            ))
        except Exception as e:
            print("  {} - Error getting metadata: {}".format(obj["key"], e))

def main():
    bucket = "my-backup-bucket"
    
    # Files to backup
    files = [
        "/important/config.json",
        "/data/database.sql",
        "/logs/app.log"
    ]
    
    # Perform backup
    backup_timestamp = backup_files(bucket, files)
    
    # List existing backups
    list_backups(bucket)

main()
```

### 4. Data Processing Pipeline

```python
load("s3", "client")
load("json")
load("time")

def process_data_pipeline():
    """Process data files from one S3 bucket to another"""
    
    s3 = client()
    
    source_bucket = "raw-data"
    processed_bucket = "processed-data"
    
    # Ensure processed bucket exists
    if not s3.bucket_exists(processed_bucket):
        s3.create_bucket(processed_bucket)
    
    # List new data files (JSON files from today)
    today = time.now().format("2006-01-02")
    prefix = "data/{}".format(today)
    
    objects = s3.list_objects(source_bucket, prefix=prefix)
    
    processed_count = 0
    
    for obj in objects["contents"]:
        if not obj["key"].endswith(".json"):
            continue
        
        print("Processing:", obj["key"])
        
        # Download and parse JSON
        raw_data = s3.get_object(source_bucket, obj["key"])
        
        try:
            data = json.decode(raw_data)
        except Exception as e:
            print("Error parsing JSON:", e)
            continue
        
        # Process data (example transformation)
        processed_data = {
            "processed_at": time.now().format(time.RFC3339),
            "source_file": obj["key"],
            "record_count": len(data) if isinstance(data, list) else 1,
            "data": data
        }
        
        # Generate processed file key
        processed_key = obj["key"].replace("data/", "processed/").replace(".json", "_processed.json")
        
        # Upload processed data
        s3.put_object(
            processed_bucket,
            processed_key,
            json.encode(processed_data),
            content_type="application/json",
            metadata={
                "source-bucket": source_bucket,
                "source-key": obj["key"],
                "processed-at": time.now().format(time.RFC3339)
            }
        )
        
        processed_count = processed_count + 1
        print("  -> Processed to:", processed_key)
    
    print("Processing complete. {} files processed.".format(processed_count))

def main():
    process_data_pipeline()

main()
```

### 5. Multi-Service Configuration

```python
load("s3", "client")

def multi_service_example():
    """Example of working with multiple S3-compatible services"""
    
    # AWS S3 client
    aws_s3 = client(
        region="us-west-2",
        # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
    )
    
    # MinIO client
    minio_s3 = client(
        endpoint="http://localhost:9000",
        access_key_id="minioadmin",
        secret_access_key="minioadmin",
        region="us-east-1",
        force_path_style=True,
        use_ssl=False
    )
    
    # DigitalOcean Spaces client
    do_s3 = client(
        endpoint="https://nyc3.digitaloceanspaces.com",
        access_key_id="YOUR_DO_SPACES_KEY",
        secret_access_key="YOUR_DO_SPACES_SECRET",
        region="nyc3"
    )
    
    # Test each service
    services = [
        ("AWS S3", aws_s3, "test-aws-bucket"),
        ("MinIO", minio_s3, "test-minio-bucket"),
        ("DigitalOcean", do_s3, "test-do-bucket")
    ]
    
    for service_name, client_instance, bucket_name in services:
        print("Testing {}...".format(service_name))
        
        try:
            # Test bucket operations
            if not client_instance.bucket_exists(bucket_name):
                print("  Creating bucket:", bucket_name)
                client_instance.create_bucket(bucket_name)
            
            # Test object operations
            test_key = "test/hello.txt"
            test_content = "Hello from {}!".format(service_name)
            
            print("  Uploading test object...")
            client_instance.put_object(bucket_name, test_key, test_content)
            
            print("  Downloading test object...")
            downloaded = client_instance.get_object(bucket_name, test_key)
            
            if downloaded == test_content:
                print("  ✓ {} test passed!".format(service_name))
            else:
                print("  ✗ {} test failed - content mismatch".format(service_name))
            
            # Cleanup
            client_instance.delete_object(bucket_name, test_key)
            
        except Exception as e:
            print("  ✗ {} test failed: {}".format(service_name, e))
        
        print()

def main():
    multi_service_example()

main()
```

### 6. Error Handling and Validation

```python
load("s3", "client", "validate_bucket_name", "validate_object_key")

def safe_s3_operations():
    """Example of robust S3 operations with error handling"""
    
    s3 = client()
    
    def safe_upload(bucket, key, content):
        """Safely upload with validation and error handling"""
        
        # Validate inputs
        if not validate_bucket_name(bucket):
            fail("Invalid bucket name: {}".format(bucket))
        
        if not validate_object_key(key):
            fail("Invalid object key: {}".format(key))
        
        if content == None or content == "":
            fail("Content cannot be empty")
        
        # Check if bucket exists
        if not s3.bucket_exists(bucket):
            print("Bucket {} does not exist, creating...".format(bucket))
            try:
                s3.create_bucket(bucket)
            except Exception as e:
                fail("Failed to create bucket: {}".format(e))
        
        # Perform upload
        try:
            s3.put_object(bucket, key, content)
            print("Successfully uploaded s3://{}/{}".format(bucket, key))
        except Exception as e:
            fail("Upload failed: {}".format(e))
    
    def safe_download(bucket, key):
        """Safely download with validation"""
        
        # Check if object exists
        if not s3.object_exists(bucket, key):
            print("Object s3://{}/{} does not exist".format(bucket, key))
            return None
        
        # Get object info first
        try:
            info = s3.get_object_info(bucket, key)
            size_mb = info["size"] / (1024 * 1024)
            
            # Check size limit (100MB)
            if size_mb > 100:
                fail("File too large: {:.2f}MB (max 100MB)".format(size_mb))
            
            print("Downloading {:.2f}MB file...".format(size_mb))
            
        except Exception as e:
            fail("Failed to get object info: {}".format(e))
        
        # Download
        try:
            content = s3.get_object(bucket, key)
            print("Successfully downloaded {} bytes".format(len(content)))
            return content
        except Exception as e:
            fail("Download failed: {}".format(e))
    
    def cleanup_old_objects(bucket, prefix, days_old=7):
        """Delete objects older than specified days"""
        
        try:
            objects = s3.list_objects(bucket, prefix=prefix)
            cutoff_time = time.now().add(-days_old * 24 * time.hour)
            
            old_objects = []
            for obj in objects["contents"]:
                # Parse object modification time
                if obj["last_modified"] < cutoff_time:
                    old_objects.append(obj["key"])
            
            if len(old_objects) == 0:
                print("No old objects to delete")
                return
            
            print("Deleting {} old objects...".format(len(old_objects)))
            result = s3.delete_objects(bucket, old_objects)
            
            print("Deleted {} objects".format(len(result["deleted"])))
            
            if "errors" in result and len(result["errors"]) > 0:
                print("Errors occurred:")
                for error in result["errors"]:
                    print("  {}: {}".format(error["key"], error["message"]))
        
        except Exception as e:
            print("Cleanup failed: {}".format(e))
    
    # Demo safe operations
    bucket_name = "safe-operations-test"
    
    # Safe upload
    safe_upload(bucket_name, "test/safe-upload.txt", "This is a safe upload test")
    
    # Safe download
    content = safe_download(bucket_name, "test/safe-upload.txt")
    if content != None:
        print("Downloaded content:", content)
    
    # Cleanup demo (commented out for safety)
    # cleanup_old_objects(bucket_name, "test/", days_old=30)

def main():
    safe_s3_operations()

main()
```

## Implementation Structure

```
s3/
├── s3.go           # Main module implementation and client creation
├── client.go       # S3 client wrapper and lifecycle management
├── bucket.go       # Bucket operations (create, delete, list, etc.)
├── object.go       # Object operations (put, get, delete, list, etc.)
├── multipart.go    # Multi-part upload handling
├── metadata.go     # Metadata and tagging operations
├── presign.go      # Pre-signed URL generation
├── utils.go        # Utility functions and validation
├── errors.go       # Error types and handling
├── s3_test.go      # Unit tests
├── example_test.go # Integration tests and examples
├── README.md       # User documentation
├── go.mod
└── go.sum
```

### Core Components

#### 1. Client Structure

```go
type S3Client struct {
    config      *Config
    awsClient   *s3.Client
    mu          sync.RWMutex
    closed      atomic.Bool
}
```

#### 2. Configuration System

Using the base package pattern for type-safe configuration:

```go
type Config struct {
    // Authentication
    AccessKeyID     *base.ConfigOption[string]       // AWS access key ID
    SecretAccessKey *base.ConfigOption[base.Secret]  // AWS secret key (secure)
    SessionToken    *base.ConfigOption[string]       // Temporary session token
    
    // Service configuration
    Region          *base.ConfigOption[string]       // AWS region
    Endpoint        *base.ConfigOption[string]       // Custom endpoint URL
    ForcePathStyle  *base.ConfigOption[bool]         // Use path-style addressing
    UseSSL          *base.ConfigOption[bool]         // Enable/disable SSL
    
    // Performance and reliability
    Timeout         *base.ConfigOption[int]          // Request timeout (seconds)
    MaxRetries      *base.ConfigOption[int]          // Maximum retry attempts
    PartSize        *base.ConfigOption[int64]        // Multi-part upload part size
    Concurrency     *base.ConfigOption[int]          // Concurrent uploads/downloads
    
    // Advanced options
    EnableLogging   *base.ConfigOption[bool]         // Enable request logging
    UserAgent       *base.ConfigOption[string]       // Custom user agent
}
```

#### 3. Response Structures

```go
type BucketInfo struct {
    Name         string    `json:"name"`
    CreationDate time.Time `json:"creation_date"`
    Region       string    `json:"region,omitempty"`
}

type ObjectInfo struct {
    Key          string            `json:"key"`
    Size         int64             `json:"size"`
    LastModified time.Time         `json:"last_modified"`
    ETag         string            `json:"etag"`
    ContentType  string            `json:"content_type,omitempty"`
    Metadata     map[string]string `json:"metadata,omitempty"`
}

type ListObjectsResult struct {
    Contents        []ObjectInfo `json:"contents"`
    CommonPrefixes  []string     `json:"common_prefixes,omitempty"`
    IsTruncated     bool         `json:"is_truncated"`
    NextMarker      string       `json:"next_marker,omitempty"`
    MaxKeys         int          `json:"max_keys"`
    Prefix          string       `json:"prefix,omitempty"`
    Delimiter       string       `json:"delimiter,omitempty"`
}
```

### Environment Variable Configuration

```bash
# Authentication
export S3_ACCESS_KEY_ID="YOUR_ACCESS_KEY"
export S3_SECRET_ACCESS_KEY="YOUR_SECRET_KEY"
export S3_SESSION_TOKEN="YOUR_SESSION_TOKEN"

# Service configuration
export S3_REGION="us-east-1"
export S3_ENDPOINT="https://s3.amazonaws.com"
export S3_FORCE_PATH_STYLE="false"
export S3_USE_SSL="true"

# Performance settings
export S3_TIMEOUT="30"
export S3_MAX_RETRIES="3"
export S3_PART_SIZE="5242880"      # 5MB
export S3_CONCURRENCY="3"

# Debug settings
export S3_ENABLE_LOGGING="false"
export S3_USER_AGENT="starlark-s3/1.0"
```

## Development Plan

### Phase 1: Core Infrastructure (Week 1)

**Priority**: Critical  
**Effort**: 25-30 hours

#### Deliverables

- Basic client creation with configuration system
- Essential bucket operations (create, list, delete, exists)
- Core object operations (put, get, delete)
- Base package integration
- Error handling framework

#### Success Criteria

```python
load("s3", "client")

s3 = client(region="us-east-1")
s3.create_bucket("test-bucket")
s3.put_object("test-bucket", "hello.txt", "Hello, World!")
content = s3.get_object("test-bucket", "hello.txt")
print(content)  # "Hello, World!"
```

### Phase 2: Object Management (Week 2)

**Priority**: High  
**Effort**: 20-25 hours

#### Deliverables

- Advanced object operations (copy, move, list with options)
- Object metadata and properties
- File upload/download operations
- Validation utilities

#### Success Criteria

```python
s3.put_object_from_file("bucket", "image.jpg", "/path/to/image.jpg")
s3.copy_object("bucket1", "file.txt", "bucket2", "copy.txt")
objects = s3.list_objects("bucket", prefix="photos/", delimiter="/")
```

### Phase 3: Advanced Features (Week 3)

**Priority**: High  
**Effort**: 22-28 hours

#### Deliverables

- Multi-part upload for large files
- Pre-signed URL generation
- Object tagging and metadata management
- Batch operations

#### Success Criteria

```python
upload_id = s3.create_multipart_upload("bucket", "large-file.zip")
# ... upload parts ...
s3.complete_multipart_upload("bucket", "large-file.zip", upload_id, parts)

url = s3.presign_url("bucket", "private.pdf", expires_in=3600)
s3.set_object_tags("bucket", "file.txt", {"env": "prod"})
```

### Phase 4: Multi-Service Support (Week 4)

**Priority**: Medium  
**Effort**: 18-22 hours

#### Deliverables

- S3-compatible service configurations
- Service-specific optimizations
- Comprehensive testing with MinIO
- Performance optimizations

#### Success Criteria

```python
# MinIO support
minio = client(
    endpoint="http://localhost:9000",
    force_path_style=True,
    use_ssl=False
)

# DigitalOcean Spaces support
do_spaces = client(
    endpoint="https://nyc3.digitaloceanspaces.com",
    region="nyc3"
)
```

### Phase 5: Polish & Documentation (Week 5)

**Priority**: Medium  
**Effort**: 15-20 hours

#### Deliverables

- Comprehensive documentation and examples
- Performance benchmarking
- Integration test suite
- Example applications

## Testing Strategy

### 1. Unit Tests (`s3_test.go`)

- Configuration parsing and validation
- Request/response handling
- Error cases and edge conditions
- Utility functions

### 2. Integration Tests (`example_test.go`)

- Real S3 operations with MinIO
- Multi-service compatibility
- Large file handling
- Error scenarios

### 3. Performance Tests

- Multi-part upload efficiency
- Concurrent operation handling
- Memory usage optimization
- Large dataset operations

### 4. Example Tests

- Complete application scenarios
- Best practice demonstrations  
- Cross-service compatibility

## Security Considerations

### 1. Credential Management

- Never log or expose credentials in error messages
- Use `base.Secret` type for sensitive configuration
- Support AWS credential chain (environment, config, IAM roles)
- Automatic credential rotation support

### 2. Input Validation

- Validate bucket names according to AWS rules
- Sanitize object keys to prevent injection
- Size limits for uploads and downloads
- Content-type validation

### 3. Network Security

- Enforce HTTPS by default
- Support custom CA certificates
- Request signing with AWS Signature Version 4
- Timeout protection against slow operations

### 4. Access Control

- Support bucket policies
- ACL management
- Server-side encryption options
- Client-side encryption capabilities

## Performance Optimizations

### 1. Connection Management

- HTTP connection pooling and reuse
- Keep-alive connections for multiple requests
- DNS caching for frequently accessed endpoints
- Circuit breaker pattern for failing services

### 2. Upload/Download Optimization

- Concurrent multi-part uploads
- Streaming for large files to minimize memory usage
- Compression for compatible content types
- Resume capability for interrupted transfers

### 3. Caching Strategy

- Client-side metadata caching
- Response caching for list operations
- Conditional requests with ETags
- Intelligent retry with exponential backoff

## Compatibility Matrix

| Service | Supported | Configuration | Notes |
|---------|-----------|---------------|-------|
| **AWS S3** | ✅ Full | Default | Complete S3 API support |
| **MinIO** | ✅ Full | `force_path_style=True` | Local/private cloud storage |
| **DigitalOcean Spaces** | ✅ Full | Custom endpoint | CDN integration available |
| **Backblaze B2** | ✅ Core | S3-compatible API | Some features limited |
| **Wasabi** | ✅ Full | Custom endpoint | Hot storage optimized |
| **Google Cloud Storage** | ✅ Core | Interoperability API | XML API compatibility |
| **IBM Cloud Object Storage** | ✅ Core | Custom endpoint | Enterprise features |
| **Alibaba Cloud OSS** | ✅ Core | Custom endpoint | Regional availability |

## Error Handling Strategy

All errors in Starlark use `fail()` since there's no try/except:

```python
def safe_operation(s3, bucket, key):
    # Input validation
    if bucket == None or bucket == "":
        fail("Bucket name cannot be empty")
    
    if key == None or key == "":
        fail("Object key cannot be empty")
    
    # Check preconditions
    if not s3.bucket_exists(bucket):
        fail("Bucket '{}' does not exist".format(bucket))
    
    # Perform operation with error handling
    try:
        return s3.get_object(bucket, key)
    except Exception as e:
        # This will be caught by the Go implementation
        fail("Failed to get object s3://{}/{}: {}".format(bucket, key, e))
```

## Dependencies

- **AWS SDK for Go v2**: Core S3 operations and authentication
- **Base Package**: Type-safe configuration and secrets management
- **Standard Library**: HTTP client, JSON, time handling
- **Sync Package**: Concurrent operations and thread safety

## Success Metrics

### 1. Performance Targets

- **Throughput**: 1000+ small operations/second
- **Latency**: <100ms for metadata operations
- **Memory**: <50MB for typical workloads
- **Large Files**: Stream 1GB+ files without memory issues

### 2. Reliability Targets

- **Error Handling**: Clear, actionable error messages
- **Retry Logic**: Automatic retry with exponential backoff
- **Network Resilience**: Handle temporary network failures
- **Data Integrity**: Verify uploads with checksums

### 3. Usability Targets

- **API Simplicity**: Intuitive function names and parameters
- **Documentation**: Comprehensive examples for all features
- **Compatibility**: Work seamlessly with existing Starlark modules
- **Migration**: Easy adoption for users of other S3 libraries

This comprehensive plan provides a solid foundation for implementing a production-ready S3 module for Starlark that follows best practices and integrates seamlessly with the existing ecosystem.
