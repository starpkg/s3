# 🗂️ S3 - Simple Storage Service Operations for Starlark

A powerful Starlark module for interacting with S3-compatible storage services including Amazon S3, MinIO, DigitalOcean Spaces, and more.

## Features

- 🔐 **Multiple Authentication Methods** - Support for access keys, environment variables, and IAM roles
- 🪣 **Comprehensive Bucket Operations** - Create, delete, list, and manage bucket configurations
- 📁 **Full Object Management** - Upload, download, copy, move, and delete objects with ease
- 🏷️ **Metadata & Tagging** - Handle custom metadata and object tags
- 🔗 **Pre-signed URLs** - Generate temporary access links for private objects
- 📦 **Multi-part Uploads** - Efficiently handle large file uploads
- 🌍 **Multi-Service Support** - Works with AWS S3, MinIO, DigitalOcean Spaces, and other S3-compatible services
- ⚡ **High Performance** - Optimized for speed with streaming and concurrent operations

## Installation

```go
go get github.com/1set/starpkg/s3
```

## Quick Start

```python
load("s3", "client")

# Create a client
s3 = client(
    access_key_id="YOUR_ACCESS_KEY",
    secret_access_key="YOUR_SECRET_KEY",
    region="us-east-1"
)

# Upload an object
s3.put_object("my-bucket", "hello.txt", "Hello, World!")

# Download an object
content = s3.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"

# List objects
objects = s3.list_objects("my-bucket")
for obj in objects["contents"]:
    print(obj["key"], obj["size"])
```

## Configuration

The S3 module supports various configuration options:

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `access_key_id` | string | AWS access key ID | Environment: `AWS_ACCESS_KEY_ID` |
| `secret_access_key` | string | AWS secret access key | Environment: `AWS_SECRET_ACCESS_KEY` |
| `region` | string | AWS region | `us-east-1` |
| `endpoint` | string | Custom endpoint for S3-compatible services | AWS S3 endpoint |
| `force_path_style` | bool | Use path-style addressing (required for MinIO) | `false` |
| `use_ssl` | bool | Enable SSL/TLS | `true` |
| `session_token` | string | Temporary session token | None |
| `timeout` | int | Request timeout in seconds | `30` |
| `max_retries` | int | Maximum retry attempts | `3` |

## Usage Examples

### Working with Different S3 Services

```python
load("s3", "client")

# AWS S3
aws_s3 = client(
    access_key_id="AWS_KEY",
    secret_access_key="AWS_SECRET",
    region="us-west-2"
)

# MinIO
minio_s3 = client(
    endpoint="http://localhost:9000",
    access_key_id="minioadmin",
    secret_access_key="minioadmin",
    force_path_style=True,
    use_ssl=False
)

# DigitalOcean Spaces
do_s3 = client(
    endpoint="https://nyc3.digitaloceanspaces.com",
    access_key_id="DO_SPACES_KEY",
    secret_access_key="DO_SPACES_SECRET",
    region="nyc3"
)
```

### Bucket Management

```python
# Create a bucket
s3.create_bucket("my-new-bucket")
s3.create_bucket("european-bucket", region="eu-west-1")

# List all buckets
buckets = s3.list_buckets()
for bucket in buckets:
    print("Bucket: {}, Created: {}".format(
        bucket["name"], 
        bucket["creation_date"]
    ))

# Check if bucket exists
if s3.bucket_exists("my-bucket"):
    print("Bucket exists!")
else:
    s3.create_bucket("my-bucket")

# Delete a bucket
s3.delete_bucket("old-bucket")
# Force delete non-empty bucket
s3.delete_bucket("full-bucket", force=True)

# Get bucket location
location = s3.get_bucket_location("my-bucket")
print("Bucket region:", location)
```

### Object Operations

```python
# Upload text
s3.put_object("my-bucket", "notes.txt", "Important notes here")

# Upload JSON with content type
data = {"user": "john", "score": 100}
s3.put_object(
    "my-bucket", 
    "data.json", 
    json.encode(data),
    content_type="application/json"
)

# Upload from file
s3.put_object_from_file(
    "my-bucket", 
    "photo.jpg", 
    "/path/to/local/photo.jpg"
)

# Download to string
content = s3.get_object("my-bucket", "notes.txt")
print(content)

# Download to file
s3.get_object_to_file(
    "my-bucket", 
    "photo.jpg", 
    "/path/to/download/photo.jpg"
)

# Get object information
info = s3.get_object_info("my-bucket", "photo.jpg")
print("Size: {} bytes".format(info["size"]))
print("Last modified: {}".format(info["last_modified"]))
print("ETag: {}".format(info["etag"]))
```

### Working with Metadata

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
```

### Object Tagging

```python
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

### Listing Objects

```python
# List all objects
objects = client.list_objects("my-bucket")
for obj in objects:
    print("{} ({} bytes)".format(obj["key"], obj["size"]))

# List with prefix
photos = client.list_objects("my-bucket", prefix="photos/2024/")
for photo in photos:
    print(photo["key"])

# List with delimiter (folder-like listing)
folders = client.list_objects("my-bucket", prefix="", delimiter="/")
for item in folders:
    if item.get("is_prefix"):
        print("Folder:", item["prefix"])
    else:
        print("File:", item["key"])
```

### Copying and Moving Objects

```python
# Copy within same bucket
client.copy_object(
    "my-bucket", "original.txt",
    "my-bucket", "copy.txt"
)

# Copy across buckets
client.copy_object(
    "source-bucket", "data.csv",
    "backup-bucket", "data-backup.csv"
)

# Move object (copy + delete)
client.move_object(
    "temp-bucket", "processing.tmp",
    "final-bucket", "processed.dat"
)
```

### Pre-signed URLs

```python
# Generate pre-signed URL for download (1 hour expiry)
download_url = client.presign_url(
    "private-bucket", 
    "confidential.pdf",
    expires_in=3600
)
print("Download link:", download_url)

# Generate pre-signed URL for upload
upload_url = client.presign_put_url(
    "upload-bucket",
    "user-upload.jpg",
    expires_in=1800,  # 30 minutes
    content_type="image/jpeg"
)
print("Upload to:", upload_url)
```

### Multi-part Upload for Large Files

```python
# Initiate multi-part upload
upload_id = client.create_multipart_upload(
    "backup-bucket",
    "large-backup.tar.gz"
)

# Upload parts
parts = []
chunk_size = 5 * 1024 * 1024  # 5MB chunks

# Read and upload file in chunks
with open("large-file.tar.gz", "rb") as f:
    part_number = 1
    while True:
        data = f.read(chunk_size)
        if not data:
            break
        
        part = client.upload_part(
            "backup-bucket",
            "large-backup.tar.gz",
            upload_id,
            part_number,
            data
        )
        parts.append(part)
        part_number += 1

# Complete the upload
client.complete_multipart_upload(
    "backup-bucket",
    "large-backup.tar.gz",
    upload_id,
    parts
)

# Or abort if something goes wrong
# client.abort_multipart_upload("backup-bucket", "large-backup.tar.gz", upload_id)
```

### Batch Operations

```python
# Delete multiple objects
objects_to_delete = [
    "temp1.txt",
    "temp2.txt", 
    "old/data.csv",
    "cache/file.tmp"
]
client.delete_objects("my-bucket", objects_to_delete)

# Copy multiple objects
files = client.list_objects("source-bucket", prefix="2024/")
for file in files:
    new_key = file["key"].replace("2024/", "archive/2024/")
    client.copy_object(
        "source-bucket", file["key"],
        "archive-bucket", new_key
    )
```

### Error Handling

```python
load("s3", "Client")

client = Client()

# Handle specific errors
def safe_upload(bucket, key, content):
    if not client.bucket_exists(bucket):
        print("Creating bucket:", bucket)
        client.create_bucket(bucket)
    
    try:
        client.put_object(bucket, key, content)
        print("Upload successful")
    except Exception as e:
        fail("Upload failed: {}".format(e))

# Validate before operations
def download_if_exists(bucket, key):
    try:
        info = client.get_object_info(bucket, key)
        if info["size"] > 100 * 1024 * 1024:  # 100MB
            fail("File too large to download: {} bytes".format(info["size"]))
        return client.get_object(bucket, key)
    except Exception as e:
        print("Object not found:", e)
        return None
```

## Complete Example: Static Website Deployment

```python
load("s3", "Client")
load("file", "read", "exists")
load("path", "join", "base")

def deploy_website(bucket_name, local_dir):
    """Deploy a static website to S3"""
    
    client = Client()
    
    # Ensure bucket exists
    if not client.bucket_exists(bucket_name):
        print("Creating bucket:", bucket_name)
        client.create_bucket(bucket_name)
    
    # Define content types
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
        ".ico": "image/x-icon"
    }
    
    # Upload all files
    uploaded = 0
    for root, dirs, files in os.walk(local_dir):
        for file in files:
            local_path = join(root, file)
            
            # Calculate S3 key (relative path)
            key = local_path[len(local_dir):].lstrip("/")
            
            # Determine content type
            ext = path.ext(file)
            content_type = content_types.get(ext, "application/octet-stream")
            
            # Upload file
            print("Uploading: {} -> s3://{}/{}".format(local_path, bucket_name, key))
            client.put_object_from_file(
                bucket_name,
                key,
                local_path,
                content_type=content_type
            )
            uploaded += 1
    
    print("Deployed {} files to S3".format(uploaded))
    
    # Generate index URL
    url = client.presign_url(bucket_name, "index.html", expires_in=3600)
    print("Website URL:", url)

# Deploy the website
deploy_website("my-website-bucket", "./dist")
```

## Best Practices

1. **Use Environment Variables for Credentials**

   ```python
   # Let the client read from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
   client = Client()
   ```

2. **Always Check Bucket Existence**

   ```python
   if not client.bucket_exists("my-bucket"):
       client.create_bucket("my-bucket")
   ```

3. **Use Content Types for Web Assets**

   ```python
   client.put_object(
       "web-bucket", 
       "index.html", 
       html_content,
       content_type="text/html"
   )
   ```

4. **Handle Large Files with Multi-part Upload**

   ```python
   # For files larger than 100MB, use multi-part upload
   if file_size > 100 * 1024 * 1024:
       # Use multi-part upload
   ```

5. **Set Appropriate Timeouts**

   ```python
   # For large file operations
   client = Client(timeout=300)  # 5 minutes
   ```

## API Reference

### Client Creation

```python
client(
    access_key_id=None,      # AWS access key
    secret_access_key=None,  # AWS secret key
    region="us-east-1",      # AWS region
    endpoint=None,           # Custom endpoint
    force_path_style=False,  # Path-style addressing
    use_ssl=True,           # Enable SSL
    session_token=None,      # Temporary credentials
    timeout=30,             # Request timeout
    max_retries=3           # Retry attempts
) -> S3Client
```

### Bucket Methods

- `create_bucket(name, region=None)` - Create a new bucket
- `delete_bucket(name, force=False)` - Delete a bucket
- `list_buckets()` - List all buckets
- `bucket_exists(name)` - Check if bucket exists
- `get_bucket_location(name)` - Get bucket region
- `set_bucket_versioning(name, enabled)` - Configure versioning

### Object Methods

- `put_object(bucket, key, content, **kwargs)` - Upload object
- `put_object_from_file(bucket, key, file_path, **kwargs)` - Upload from file
- `get_object(bucket, key)` - Download object
- `get_object_to_file(bucket, key, file_path)` - Download to file
- `delete_object(bucket, key)` - Delete single object
- `delete_objects(bucket, keys)` - Delete multiple objects
- `list_objects(bucket, prefix="", delimiter="")` - List objects
- `copy_object(src_bucket, src_key, dst_bucket, dst_key)` - Copy object
- `move_object(src_bucket, src_key, dst_bucket, dst_key)` - Move object
- `get_object_info(bucket, key)` - Get object metadata
- `get_object_metadata(bucket, key)` - Get custom metadata
- `set_object_tags(bucket, key, tags)` - Set object tags
- `get_object_tags(bucket, key)` - Get object tags
- `presign_url(bucket, key, expires_in=3600)` - Generate pre-signed URL

### Multi-part Upload Methods

- `create_multipart_upload(bucket, key, **kwargs)` - Start multi-part upload
- `upload_part(bucket, key, upload_id, part_number, content)` - Upload part
- `complete_multipart_upload(bucket, key, upload_id, parts)` - Complete upload
- `abort_multipart_upload(bucket, key, upload_id)` - Abort upload

## Contributing

PRs are welcome! Please ensure all tests pass and add new tests for any new functionality.

## License

MIT License - see LICENSE file for details
