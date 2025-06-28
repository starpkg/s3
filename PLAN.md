# S3 Starlark Module Development Plan

## 🗂️ S3 Module - Simple Storage Service Operations for Starlark

**Module Name**: `s3`  
**Emoji**: 🗂️  
**Description**: Complete S3-compatible storage operations for Starlark  
**Tagline**: Unified interface for Amazon S3, Cloudflare R2, and all S3-compatible storage services

## Key Features

- 🔐 **Multiple Authentication Methods** - Support for access keys, environment variables, and IAM roles
- 🪣 **Comprehensive Bucket Operations** - Create, delete, list, and manage bucket configurations
- 📁 **Full Object Management** - Upload, download, copy, move, and delete objects with ease
- 🏷️ **Metadata & Tagging** - Handle custom metadata and object tags
- 🔗 **Pre-signed URLs** - Generate temporary access links for private objects
- 📦 **Multi-part Uploads** - Efficiently handle large file uploads
- 🌍 **Multi-Service Support** - Works with Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, and MinIO
- ⚡ **High Performance** - Optimized for speed with streaming and concurrent operations

## Supported S3-Compatible Services

| Service | Status | Notes |
|---------|--------|-------|
| **Amazon S3** | ✅ Full Support | Reference implementation with all features |
| **Cloudflare R2** | ✅ Full Support | S3-compatible API with zero egress fees |
| **Backblaze B2** | ✅ Full Support | Cost-effective storage with S3-compatible API |
| **DigitalOcean Spaces** | ✅ Full Support | Developer-friendly S3-compatible storage |
| **MinIO** | ✅ Full Support | Self-hosted S3-compatible object storage |

## Executive Summary

The `s3` module provides comprehensive S3-compatible storage operations for Starlark scripts. It focuses on simplicity, security, and performance while supporting major S3-compatible services including Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, and MinIO. The design emphasizes ease of use with powerful features for both simple scripts and complex applications.

## Core Design Principles

1. **Function-based API**: Uses `create_client()` function instead of class constructors
2. **S3-compatible First**: Works seamlessly with any S3-compatible service
3. **Security by Default**: Secure credential handling with base package integration
4. **High Performance**: Optimized for large files with streaming and concurrent operations
5. **Starlark Native**: Designed specifically for Starlark constraints and patterns
6. **Production Ready**: Built for reliability with proper error handling and retries

## Starlark Constraints & Adaptations

### Key Limitations Addressed

- ❌ **No Classes**: Use `create_client()` function returning object with methods
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

#### create_client()

Create an S3-compatible client for storage operations.

```python
create_client(
    service_type="auto",        # Service type: "aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean", "minio", "auto"
    endpoint=None,              # Custom endpoint URL for S3-compatible services
    aws_region="us-east-1",     # AWS region or service region
    aws_access_key=None,        # Access key ID (uses AWS_ACCESS_KEY_ID env var if None)
    aws_secret_key=None,        # Secret access key (uses AWS_SECRET_ACCESS_KEY env var if None)
    aws_session_token=None,     # Session token for temporary credentials (optional)
    force_path_style=False,     # Use path-style addressing (required for MinIO and some services)
    use_ssl=True,               # Enable/disable SSL/TLS
    timeout=30,                 # Request timeout in seconds
    max_retries=3,              # Maximum retry attempts for failed requests
    part_size=5242880,          # Multi-part upload part size in bytes (5MB default)
    concurrency=3,              # Number of concurrent uploads/downloads
    enable_logging=False,       # Enable request/response logging for debugging
    user_agent="starlark-s3/1.0", # Custom User-Agent header
    **config                    # Additional service-specific configuration options
) -> S3Client
```

**Parameters:**
- `service_type` (str): Target service type. Auto-detects if not specified.
- `endpoint` (str, optional): Custom endpoint URL. Auto-configured for known services.
- `aws_region` (str): Service region. Defaults to "us-east-1".
- `aws_access_key` (str, optional): Access key ID. Uses environment variable if not provided.
- `aws_secret_key` (str, optional): Secret access key. Uses environment variable if not provided.
- `aws_session_token` (str, optional): Temporary session token for STS credentials.
- `force_path_style` (bool): Use path-style URLs instead of virtual-hosted style.
- `use_ssl` (bool): Enable SSL/TLS encryption. Defaults to True.
- `timeout` (int): Request timeout in seconds.
- `max_retries` (int): Maximum number of retry attempts for failed requests.
- `part_size` (int): Size of each part in multi-part uploads (bytes).
- `concurrency` (int): Number of concurrent operations allowed.
- `enable_logging` (bool): Enable detailed request logging.
- `user_agent` (str): Custom User-Agent string for requests.

**Returns:**
- `S3Client`: Configured S3 client object with methods for storage operations.

**Example:**
```python
load("s3", "create_client")

# Create client with environment variables
s3 = create_client()

# Create client with explicit credentials
s3 = create_client(
    service_type="aws_s3",
    aws_access_key="AKIAIOSFODNN7EXAMPLE",
    aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    aws_region="us-west-2"
)
```

#### Utility Functions

##### parse_s3_url()

Parse an S3 URL into bucket and key components.

```python
parse_s3_url(url) -> dict
```

**Parameters:**
- `url` (str): S3 URL in format "s3://bucket/key" or "https://bucket.s3.amazonaws.com/key"

**Returns:**
- `dict`: Dictionary with "bucket" and "key" fields

**Example:**
```python
result = parse_s3_url("s3://my-bucket/path/to/file.txt")
print(result["bucket"])  # "my-bucket"
print(result["key"])     # "path/to/file.txt"
```

##### generate_s3_url()

Generate a standard S3 URL from bucket and key.

```python
generate_s3_url(bucket, key, region="us-east-1") -> str
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key/path
- `region` (str, optional): AWS region for URL generation

**Returns:**
- `str`: Standard S3 URL

**Example:**
```python
url = generate_s3_url("my-bucket", "path/file.txt", "us-west-2")
print(url)  # "https://my-bucket.s3.us-west-2.amazonaws.com/path/file.txt"
```

##### validate_bucket_name()

Validate bucket name according to S3 naming rules.

```python
validate_bucket_name(name) -> bool
```

**Parameters:**
- `name` (str): Bucket name to validate

**Returns:**
- `bool`: True if bucket name is valid, False otherwise

**Example:**
```python
if validate_bucket_name("my-valid-bucket"):
    print("Bucket name is valid")
```

##### validate_object_key()

Validate object key/path according to S3 key rules.

```python
validate_object_key(key) -> bool
```

**Parameters:**
- `key` (str): Object key to validate

**Returns:**
- `bool`: True if object key is valid, False otherwise

**Example:**
```python
if validate_object_key("path/to/my-file.txt"):
    print("Object key is valid")
```

##### get_supported_services()

Get list of supported S3-compatible services.

```python
get_supported_services() -> list
```

**Returns:**
- `list`: List of supported service type strings

**Example:**
```python
services = get_supported_services()
print("Supported services:", services)
# ["aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean", "minio"]
```

##### get_client_info()

Get information about a configured S3 client.

```python
get_client_info(client) -> dict
```

**Parameters:**
- `client` (S3Client): S3 client object

**Returns:**
- `dict`: Client configuration information (excludes sensitive data)

**Example:**
```python
info = get_client_info(s3)
print("Service type:", info["service_type"])
print("Region:", info["region"])
print("Endpoint:", info["endpoint"])
```

### Service-Specific Client Examples

#### Amazon S3

```python
load("s3", "create_client")

# Using environment variables (recommended)
s3 = create_client(service_type="aws_s3", aws_region="us-east-1")

# With explicit credentials
s3 = create_client(
    service_type="aws_s3",
    aws_access_key="AKIAIOSFODNN7EXAMPLE",
    aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    aws_region="us-east-1"
)
```

#### Cloudflare R2

```python
# Cloudflare R2 with custom endpoint
s3 = create_client(
    service_type="cloudflare_r2",
    endpoint="https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com",
    aws_access_key="YOUR_R2_ACCESS_KEY",
    aws_secret_key="YOUR_R2_SECRET_KEY",
    aws_region="auto"  # R2 uses "auto" region
)
```

#### Backblaze B2

```python
# Backblaze B2 S3-compatible API
s3 = create_client(
    service_type="backblaze_b2",
    endpoint="https://s3.us-west-004.backblazeb2.com",
    aws_access_key="YOUR_KEY_ID",
    aws_secret_key="YOUR_APPLICATION_KEY",
    aws_region="us-west-004"
)
```

#### DigitalOcean Spaces

```python
# DigitalOcean Spaces
s3 = create_client(
    service_type="digitalocean",
    endpoint="https://nyc3.digitaloceanspaces.com",
    aws_access_key="YOUR_SPACES_KEY",
    aws_secret_key="YOUR_SPACES_SECRET",
    aws_region="nyc3"
)
```

#### MinIO (Self-hosted)

```python
# MinIO local development
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    aws_access_key="minioadmin",
    aws_secret_key="minioadmin",
    aws_region="us-east-1",
    force_path_style=True,  # Required for MinIO
    use_ssl=False  # For local development
)
```

#### Auto-Detection

```python
# Auto-detect service from environment variables
s3 = create_client()  # Uses AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, etc.

# With advanced configuration
s3 = create_client(
    aws_region="eu-west-1",
    timeout=60,
    max_retries=5,
    part_size=10485760,  # 10MB parts
    concurrency=5
)
```

## Configuration Options

The S3 module supports various configuration options:

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `service_type` | string | Service type ("aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean", "minio", "auto") | `auto` |
| `aws_access_key` | string | AWS access key ID | Environment: `AWS_ACCESS_KEY_ID` |
| `aws_secret_key` | string | AWS secret access key | Environment: `AWS_SECRET_ACCESS_KEY` |
| `aws_session_token` | string | AWS session token | Environment: `AWS_SESSION_TOKEN` |
| `aws_region` | string | AWS region | Environment: `AWS_DEFAULT_REGION` or `us-east-1` |
| `endpoint` | string | Custom endpoint for S3-compatible services | Auto-detected based on service_type |
| `force_path_style` | bool | Use path-style addressing (required for MinIO) | `false` |
| `use_ssl` | bool | Enable SSL/TLS | `true` |
| `timeout` | int | Request timeout in seconds | `30` |
| `max_retries` | int | Maximum retry attempts | `3` |
| `part_size` | int | Multi-part upload part size in bytes | `5242880` (5MB) |
| `concurrency` | int | Concurrent uploads/downloads | `3` |
| `enable_logging` | bool | Enable request logging | `false` |
| `user_agent` | string | Custom user agent | `starlark-s3/1.0` |

### S3Client Object API

#### Bucket Operations

##### create_bucket()

Create a new storage bucket.

```python
s3.create_bucket(name, region=None, **options) -> None
```

**Parameters:**
- `name` (str): Bucket name (must be globally unique and follow naming rules)
- `region` (str, optional): Region to create bucket in (uses client default if None)
- `**options`: Additional service-specific options

**Raises:**
- `fail()`: If bucket already exists or name is invalid

##### delete_bucket()

Delete a storage bucket.

```python
s3.delete_bucket(name, force=False) -> None
```

**Parameters:**
- `name` (str): Bucket name to delete
- `force` (bool): If True, delete bucket even if it contains objects

**Raises:**
- `fail()`: If bucket doesn't exist or deletion fails

##### list_buckets()

List all accessible buckets.

```python
s3.list_buckets() -> list
```

**Returns:**
- `list`: List of bucket dictionaries with "name", "creation_date", and "region" fields

##### bucket_exists()

Check if a bucket exists and is accessible.

```python
s3.bucket_exists(name) -> bool
```

**Parameters:**
- `name` (str): Bucket name to check

**Returns:**
- `bool`: True if bucket exists and is accessible

##### get_bucket_location()

Get the region/location of a bucket.

```python
s3.get_bucket_location(name) -> str
```

**Parameters:**
- `name` (str): Bucket name

**Returns:**
- `str`: Bucket region/location

##### set_bucket_versioning()

Enable or disable bucket versioning.

```python
s3.set_bucket_versioning(name, enabled=True) -> None
```

**Parameters:**
- `name` (str): Bucket name
- `enabled` (bool): Enable (True) or disable (False) versioning

##### get_bucket_versioning()

Get bucket versioning configuration.

```python
s3.get_bucket_versioning(name) -> dict
```

**Parameters:**
- `name` (str): Bucket name

**Returns:**
- `dict`: Versioning configuration with "enabled" and "status" fields

#### Object Operations - Core

##### put_object()

Upload content as an object.

```python
s3.put_object(bucket, key, content, content_type=None, metadata=None, tags=None, **options) -> None
```

**Parameters:**
- `bucket` (str): Target bucket name
- `key` (str): Object key/path
- `content` (str|bytes): Content to upload
- `content_type` (str, optional): MIME type of content
- `metadata` (dict, optional): Custom metadata key-value pairs
- `tags` (dict, optional): Object tags for classification
- `**options`: Additional upload options

##### put_object_from_file()

Upload a file as an object.

```python
s3.put_object_from_file(bucket, key, file_path, content_type=None, metadata=None, **options) -> None
```

**Parameters:**
- `bucket` (str): Target bucket name
- `key` (str): Object key/path
- `file_path` (str): Local file path to upload
- `content_type` (str, optional): MIME type (auto-detected if None)
- `metadata` (dict, optional): Custom metadata
- `**options`: Additional upload options

##### get_object()

Download object content to memory.

```python
s3.get_object(bucket, key) -> str
```

**Parameters:**
- `bucket` (str): Source bucket name
- `key` (str): Object key/path

**Returns:**
- `str`: Object content as string

##### get_object_to_file()

Download object content to a file.

```python
s3.get_object_to_file(bucket, key, file_path) -> None
```

**Parameters:**
- `bucket` (str): Source bucket name
- `key` (str): Object key/path
- `file_path` (str): Local file path to save content

##### delete_object()

Delete a single object.

```python
s3.delete_object(bucket, key) -> None
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key to delete

##### delete_objects()

Delete multiple objects in a batch operation.

```python
s3.delete_objects(bucket, keys) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `keys` (list): List of object keys to delete

**Returns:**
- `dict`: Result with "deleted" and "errors" lists

#### Object Operations - Advanced

##### copy_object()

Copy an object to a new location.

```python
s3.copy_object(src_bucket, src_key, dst_bucket, dst_key, metadata=None, **options) -> None
```

**Parameters:**
- `src_bucket` (str): Source bucket name
- `src_key` (str): Source object key
- `dst_bucket` (str): Destination bucket name
- `dst_key` (str): Destination object key
- `metadata` (dict, optional): New metadata for copied object
- `**options`: Additional copy options

##### move_object()

Move an object to a new location (copy + delete).

```python
s3.move_object(src_bucket, src_key, dst_bucket, dst_key, **options) -> None
```

**Parameters:**
- `src_bucket` (str): Source bucket name
- `src_key` (str): Source object key
- `dst_bucket` (str): Destination bucket name
- `dst_key` (str): Destination object key
- `**options`: Additional move options

##### list_objects()

List objects in a bucket with filtering options.

```python
s3.list_objects(bucket, prefix="", delimiter="", max_keys=1000, start_after="") -> dict
```

**Parameters:**
- `bucket` (str): Bucket name to list
- `prefix` (str, optional): Filter objects by prefix
- `delimiter` (str, optional): Delimiter for grouping (e.g., "/" for directories)
- `max_keys` (int): Maximum number of objects to return
- `start_after` (str, optional): Start listing after this key

**Returns:**
- `dict`: Result with "contents", "common_prefixes", "is_truncated", and pagination fields

##### get_object_info()

Get object metadata and properties.

```python
s3.get_object_info(bucket, key) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key

**Returns:**
- `dict`: Object information with "size", "last_modified", "etag", "content_type", and "metadata" fields

##### object_exists()

Check if an object exists.

```python
s3.object_exists(bucket, key) -> bool
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key to check

**Returns:**
- `bool`: True if object exists

#### Metadata and Tagging

##### get_object_metadata()

Get custom metadata for an object.

```python
s3.get_object_metadata(bucket, key) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key

**Returns:**
- `dict`: Custom metadata key-value pairs

##### set_object_metadata()

Set custom metadata for an object.

```python
s3.set_object_metadata(bucket, key, metadata) -> None
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `metadata` (dict): Metadata key-value pairs to set

##### get_object_tags()

Get tags for an object.

```python
s3.get_object_tags(bucket, key) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key

**Returns:**
- `dict`: Object tags as key-value pairs

##### set_object_tags()

Set tags for an object.

```python
s3.set_object_tags(bucket, key, tags) -> None
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `tags` (dict): Tags to set as key-value pairs

##### delete_object_tags()

Remove all tags from an object.

```python
s3.delete_object_tags(bucket, key) -> None
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key

#### Pre-signed URLs

##### presign_url()

Generate a pre-signed URL for object access.

```python
s3.presign_url(bucket, key, expires_in=3600, method="GET") -> str
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `expires_in` (int): URL expiration time in seconds
- `method` (str): HTTP method ("GET", "PUT", "DELETE")

**Returns:**
- `str`: Pre-signed URL

##### presign_put_url()

Generate a pre-signed URL for object upload.

```python
s3.presign_put_url(bucket, key, expires_in=3600, content_type=None, **options) -> str
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `expires_in` (int): URL expiration time in seconds
- `content_type` (str, optional): Required content type for upload
- `**options`: Additional constraints

**Returns:**
- `str`: Pre-signed upload URL

##### presign_post()

Generate pre-signed POST data for browser uploads.

```python
s3.presign_post(bucket, key, expires_in=3600, content_length_range=None, **options) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `expires_in` (int): URL expiration time in seconds
- `content_length_range` (tuple, optional): Min/max file size limits
- `**options`: Additional constraints

**Returns:**
- `dict`: POST form data with "url" and "fields"

#### Multi-part Upload

##### create_multipart_upload()

Initiate a multi-part upload.

```python
s3.create_multipart_upload(bucket, key, content_type=None, metadata=None, **options) -> str
```

**Parameters:**
- `bucket` (str): Target bucket name
- `key` (str): Object key
- `content_type` (str, optional): MIME type
- `metadata` (dict, optional): Custom metadata
- `**options`: Additional options

**Returns:**
- `str`: Upload ID for subsequent operations

##### upload_part()

Upload a part of a multi-part upload.

```python
s3.upload_part(bucket, key, upload_id, part_number, content) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `upload_id` (str): Multi-part upload ID
- `part_number` (int): Part number (1-10000)
- `content` (str|bytes): Part content

**Returns:**
- `dict`: Part information with "part_number" and "etag" fields

##### complete_multipart_upload()

Complete a multi-part upload.

```python
s3.complete_multipart_upload(bucket, key, upload_id, parts) -> dict
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `upload_id` (str): Multi-part upload ID
- `parts` (list): List of part dictionaries from upload_part()

**Returns:**
- `dict`: Upload result with "etag" and "location" fields

##### abort_multipart_upload()

Cancel a multi-part upload.

```python
s3.abort_multipart_upload(bucket, key, upload_id) -> None
```

**Parameters:**
- `bucket` (str): Bucket name
- `key` (str): Object key
- `upload_id` (str): Multi-part upload ID to cancel

##### list_multipart_uploads()

List ongoing multi-part uploads.

```python
s3.list_multipart_uploads(bucket, prefix="", max_uploads=1000) -> list
```

**Parameters:**
- `bucket` (str): Bucket name
- `prefix` (str, optional): Filter uploads by key prefix
- `max_uploads` (int): Maximum uploads to return

**Returns:**
- `list`: List of upload dictionaries with "upload_id", "key", and "initiated" fields

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

The S3 module includes comprehensive examples that demonstrate various use cases. These examples are located in the `examples/` directory and can be run directly with StarCLI.

### Example Files

#### 1. Basic File Management
**File**: `examples/basic_file_management.star`

Demonstrates core S3 operations including:
- Creating and managing buckets
- Uploading objects with metadata
- Listing and filtering objects
- Downloading content
- Generating pre-signed URLs
- Copying objects

```bash
starcli s3/examples/basic_file_management.star
```

#### 2. Website Deployment
**File**: `examples/website_deployment.star`

Shows how to deploy static websites to S3:
- Content type detection and configuration
- Cache control headers for optimization
- Batch file uploads
- Website hosting configuration
- Performance considerations

```bash
starcli s3/examples/website_deployment.star
```

#### 3. Backup System
**File**: `examples/backup_system.star`

Implements a complete backup solution:
- Automated file backups with timestamps
- Metadata and tagging for organization
- Backup listing and management
- Retention policy implementation
- Restore functionality

```bash
starcli s3/examples/backup_system.star
```

#### 4. Multi-Service Configuration
**File**: `examples/multi_service_configuration.star`

Demonstrates working with multiple S3-compatible services:
- Service-specific configurations
- Feature compatibility testing
- Performance comparisons
- Error handling across services

```bash
starcli s3/examples/multi_service_configuration.star
```

### 2. Website Static File Deployment

```python
load("s3", "connect")
load("file", "exists", "read")
load("path", "join", "ext")

def deploy_website(bucket_name, local_dir):
    """Deploy a static website to S3"""
    
    s3 = connect()
    
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
load("s3", "connect")
load("time")
load("file", "read", "exists")
load("path", "join")

def backup_files(bucket_name, files_to_backup):
    """Backup files to S3 with timestamp and metadata"""
    
    s3 = connect()
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
    
    s3 = connect()
    
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
load("s3", "connect")
load("json")
load("time")

def process_data_pipeline():
    """Process data files from one S3 bucket to another"""
    
    s3 = connect()
    
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
load("s3", "connect")

def multi_service_example():
    """Example of working with multiple S3-compatible services"""
    
    # AWS S3 client
    aws_s3 = create_client(
        service_type="aws_s3",
        aws_region="us-west-2",
        # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
    )
    
    # MinIO client
    minio_s3 = create_client(
        service_type="minio",
        endpoint="http://localhost:9000",
        aws_access_key="minioadmin",
        aws_secret_key="minioadmin",
        aws_region="us-east-1",
        force_path_style=True,
        use_ssl=False
    )
    
    # DigitalOcean Spaces client
    do_s3 = create_client(
        service_type="digitalocean",
        endpoint="https://nyc3.digitaloceanspaces.com",
        aws_access_key="YOUR_DO_SPACES_KEY",
        aws_secret_key="YOUR_DO_SPACES_SECRET",
        aws_region="nyc3"
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
load("s3", "connect", "validate_bucket_name", "validate_object_key")

def safe_s3_operations():
    """Example of robust S3 operations with error handling"""
    
    s3 = connect()
    
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
# Primary service configuration
export S3_SERVICE_TYPE="aws_s3"                    # Service type
export S3_ENDPOINT="https://s3.amazonaws.com"      # Custom endpoint
export S3_TIMEOUT="30"                             # Connection timeout
export S3_MAX_RETRIES="3"                          # Maximum retry attempts

# AWS Authentication (compatible with AWS CLI/SDK)
export AWS_ACCESS_KEY_ID="YOUR_ACCESS_KEY"         # AWS access key ID
export AWS_SECRET_ACCESS_KEY="YOUR_SECRET_KEY"     # AWS secret access key
export AWS_SESSION_TOKEN="YOUR_SESSION_TOKEN"      # AWS session token (optional)
export AWS_DEFAULT_REGION="us-east-1"              # AWS region

# S3-specific configuration
export S3_FORCE_PATH_STYLE="false"                 # Path-style addressing
export S3_USE_SSL="true"                           # Enable SSL/TLS
export S3_PART_SIZE="5242880"                      # Multipart upload part size (5MB)
export S3_CONCURRENCY="3"                          # Concurrent operations

# Debug and monitoring
export S3_ENABLE_LOGGING="false"                   # Enable request logging
export S3_USER_AGENT="starlark-s3/1.0"            # Custom user agent
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
load("s3", "create_client")

s3 = create_client(aws_region="us-east-1")
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
        minio = create_client(
            service_type="minio",
            endpoint="http://localhost:9000",
            force_path_style=True,
            use_ssl=False
        )

        # DigitalOcean Spaces support
        do_spaces = create_client(
            service_type="digitalocean",
            endpoint="https://nyc3.digitaloceanspaces.com",
            aws_region="nyc3"
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

## Service Compatibility Analysis

### Overview Matrix

| Service | Support Level | Configuration Required | Primary Use Case |
|---------|---------------|----------------------|------------------|
| **Amazon S3** | ✅ Complete | Default settings | Production cloud storage |
| **Cloudflare R2** | ✅ Complete | Custom endpoint | Zero-egress global storage |
| **Backblaze B2** | ✅ Complete | S3-compatible API | Cost-effective backup storage |
| **DigitalOcean Spaces** | ✅ Complete | Custom endpoint | Developer-friendly hosting |
| **MinIO** | ✅ Complete | `force_path_style=True` | Self-hosted/on-premises |

**Support Level Legend:**

- ✅ **Complete**: Full API compatibility with all features supported
- ⚠️ **Limited**: Core features work, some advanced features may be unavailable
- ❌ **Unsupported**: Not compatible with S3 API

### Service-Specific Configuration Details

#### Amazon S3 (Reference Implementation)
- **Endpoint**: Auto-configured based on region
- **Authentication**: Full AWS credential chain support
- **Special Features**: All S3 features including advanced IAM, KMS encryption, lifecycle policies
- **Best For**: Production workloads, enterprise applications, full AWS ecosystem integration

#### Cloudflare R2
- **Endpoint**: `https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com`
- **Region**: Use `"auto"` as the region
- **Special Features**: Zero egress fees, global edge distribution
- **Best For**: Static assets, CDN content, global applications

#### Backblaze B2
- **Endpoint**: Region-specific (e.g., `https://s3.us-west-004.backblazeb2.com`)
- **Authentication**: Application Key ID and Application Key
- **Special Features**: Cost-effective storage, good for archival
- **Best For**: Backup storage, archival, cost-sensitive applications

#### DigitalOcean Spaces
- **Endpoint**: Region-specific (e.g., `https://nyc3.digitaloceanspaces.com`)
- **Authentication**: Spaces access key and secret
- **Special Features**: Simple pricing, CDN integration available
- **Best For**: Small to medium applications, developer projects

#### MinIO
- **Endpoint**: Self-hosted (e.g., `http://localhost:9000`)
- **Configuration**: Requires `force_path_style=True`
- **Special Features**: Full S3 API compatibility, self-hosted
- **Best For**: On-premises deployments, development environments, hybrid cloud

### Authentication Methods Comparison

| Service | Access Keys | IAM Roles | STS Tokens | Environment Variables | Notes |
|---------|-------------|-----------|------------|----------------------|-------|
| **Amazon S3** | ✅ | ✅ | ✅ | ✅ | Full AWS credential chain support |
| **Cloudflare R2** | ✅ | ❌ | ❌ | ✅ | R2 API tokens and keys |
| **Backblaze B2** | ✅ | ❌ | ❌ | ✅ | Application Key ID and Key |
| **DigitalOcean Spaces** | ✅ | ❌ | ❌ | ✅ | Spaces access key and secret |
| **MinIO** | ✅ | ❌ | ❌ | ✅ | Simple access key authentication |

### Bucket Operations Support

| Operation | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Notes |
|-----------|-----------|---------------|--------------|--------------|-------|-------|
| `create_bucket()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_bucket()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `list_buckets()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `bucket_exists()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_bucket_location()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `set_bucket_versioning()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| `get_bucket_versioning()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| Regional bucket creation | ✅ | ✅ | ✅ | ✅ | ✅ | All services support regions |

### Object Operations Support

| Operation | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Notes |
|-----------|-----------|---------------|--------------|--------------|-------|-------|
| `put_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_objects()` (batch) | ✅ | ✅ | ✅ | ✅ | ✅ | Universal batch support |
| `copy_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `list_objects()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_object_info()` (HEAD) | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `object_exists()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| Range requests | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |

### Advanced Features Support

| Feature | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Implementation Notes |
|---------|-----------|---------------|--------------|--------------|-------|---------------------|
| **Multipart Upload** | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `create_multipart_upload()` | ✅ | ✅ | ✅ | ✅ | ✅ | Standard implementation |
| `upload_part()` | ✅ | ✅ | ✅ | ✅ | ✅ | Standard implementation |
| `complete_multipart_upload()` | ✅ | ✅ | ✅ | ✅ | ✅ | Standard implementation |
| `abort_multipart_upload()` | ✅ | ✅ | ✅ | ✅ | ✅ | Standard implementation |
| `list_multipart_uploads()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| **Pre-signed URLs** | ✅ | ✅ | ✅ | ✅ | ✅ | AWS signature compatible |
| `presign_url()` (GET) | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `presign_put_url()` (PUT) | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `presign_post()` (POST) | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |

### Metadata and Tagging Support

| Feature | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Limitations |
|---------|-----------|---------------|--------------|--------------|-------|-------------|
| **Custom Metadata** | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_object_metadata()` | ✅ | ✅ | ✅ | ✅ | ✅ | Header-based metadata |
| `set_object_metadata()` | ✅ | ✅ | ✅ | ✅ | ✅ | Copy operation required |
| Metadata size limits | 2KB | 2KB | 2KB | 2KB | 2KB | Standard 2KB limit |
| **Object Tagging** | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| `get_object_tags()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| `set_object_tags()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| `delete_object_tags()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| Tag count limits | 10 | N/A | N/A | N/A | 10 | 10 tags maximum |

### Configuration Parameters Support

| Parameter | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Universal | Notes |
|-----------|-----------|---------------|--------------|--------------|-------|-----------|-------|
| `aws_access_key` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Required for authentication |
| `aws_secret_key` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Required for authentication |
| `aws_region` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Universal region support |
| `endpoint` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Custom endpoint support |
| `force_path_style` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Required for MinIO |
| `use_ssl` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | SSL/TLS configuration |
| `aws_session_token` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | AWS-specific feature |
| `timeout` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Universal timeout support |
| `max_retries` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Retry configuration |
| `part_size` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Multipart upload tuning |
| `concurrency` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Concurrent operations |

### Service Configuration Examples

#### Amazon S3 (Reference Implementation)

```python
s3 = create_client(
    service_type="aws_s3",
    aws_region="us-east-1",
    # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
    timeout=30,
    max_retries=3
)
```

#### Cloudflare R2

```python
s3 = create_client(
    service_type="cloudflare_r2",
    endpoint="https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com",
    aws_access_key="YOUR_R2_ACCESS_KEY_ID",
    aws_secret_key="YOUR_R2_SECRET_ACCESS_KEY",
    aws_region="auto",  # R2 uses "auto" region
    timeout=30
)
```

#### Backblaze B2

```python
s3 = create_client(
    service_type="backblaze_b2",
    endpoint="https://s3.us-west-004.backblazeb2.com",
    aws_access_key="YOUR_B2_KEY_ID",
    aws_secret_key="YOUR_B2_APPLICATION_KEY",
    aws_region="us-west-004",
    timeout=60  # B2 can be slower
)
```

#### DigitalOcean Spaces

```python
s3 = create_client(
    service_type="digitalocean",
    endpoint="https://nyc3.digitaloceanspaces.com",
    aws_access_key="YOUR_SPACES_ACCESS_KEY",
    aws_secret_key="YOUR_SPACES_SECRET_KEY",
    aws_region="nyc3"
)
```

#### MinIO (Self-hosted)

```python
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    aws_access_key="minioadmin",
    aws_secret_key="minioadmin",
    aws_region="us-east-1",
    force_path_style=True,  # Required for MinIO
    use_ssl=False  # For local development
)
```

### Feature Compatibility Notes

#### **Universal Features** (Work with all services)

- Basic CRUD operations (create, read, update, delete)
- Bucket management (create, delete, list, exists)
- Object metadata (standard HTTP headers)
- Multipart uploads for large files
- SSL/TLS encryption in transit
- Pre-signed URLs for temporary access
- Batch operations for efficiency

#### **Amazon S3 and MinIO Exclusive Features**

- Object tagging and tag management
- STS token authentication (AWS S3 only)
- Advanced IAM integration (AWS S3 only)
- Server-side encryption with KMS (AWS S3 only)
- Bucket policies and ACLs
- Presign POST operations

#### **Service-Specific Strengths**

**Amazon S3:**
- Complete feature set and reference implementation
- Advanced security and compliance features
- Comprehensive ecosystem integration

**Cloudflare R2:**
- Zero egress fees for data transfer
- Global edge network integration
- Simple and predictable pricing

**Backblaze B2:**
- Very cost-effective storage pricing
- Good for backup and archival use cases
- Transparent pricing model

**DigitalOcean Spaces:**
- Developer-friendly interface and pricing
- Tight integration with DigitalOcean ecosystem
- CDN integration available

**MinIO:**
- Full S3 API compatibility for self-hosting
- High performance for on-premises deployments
- Kubernetes-native with operator support

### Recommended Service Selection

| Use Case | Primary Choice | Alternative | Reason |
|----------|---------------|-------------|--------|
| **Production AWS Ecosystem** | Amazon S3 | None | Full feature compatibility and AWS integration |
| **Global CDN/Edge Content** | Cloudflare R2 | Amazon S3 | Zero egress fees, global distribution |
| **Cost-Sensitive Backup** | Backblaze B2 | DigitalOcean Spaces | Very low storage costs |
| **Developer Projects** | DigitalOcean Spaces | MinIO | Simple pricing, developer-friendly |
| **On-Premises/Self-Hosted** | MinIO | None | Full S3 compatibility, self-hosted |
| **Development/Testing** | MinIO | DigitalOcean Spaces | Local development, full feature set |
| **Hybrid Cloud Strategy** | Amazon S3 + MinIO | Multiple services | AWS for production, MinIO for on-premises |

## Error Handling Strategy

### Starlark Error Patterns

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

### Service-Specific Error Handling

Different S3-compatible services return different error codes and messages. The module normalizes these into consistent Starlark failures:

| Error Scenario | AWS S3 | MinIO | Azure Blob | Normalized Starlark Error |
|----------------|--------|-------|------------|---------------------------|
| **Bucket not found** | `NoSuchBucket` | `NoSuchBucket` | `ContainerNotFound` | `"Bucket 'name' does not exist"` |
| **Object not found** | `NoSuchKey` | `NoSuchKey` | `BlobNotFound` | `"Object 's3://bucket/key' not found"` |
| **Access denied** | `AccessDenied` | `AccessDenied` | `AuthorizationFailure` | `"Permission denied for 's3://bucket/key'"` |
| **Invalid credentials** | `InvalidAccessKeyId` | `InvalidAccessKeyId` | `AuthenticationFailed` | `"Authentication failed: invalid credentials"` |
| **Bucket already exists** | `BucketAlreadyExists` | `BucketAlreadyExists` | `ContainerAlreadyExists` | `"Bucket 'name' already exists"` |
| **Invalid bucket name** | `InvalidBucketName` | `InvalidBucketName` | `InvalidResourceName` | `"Invalid bucket name: 'name'"` |
| **Network timeout** | `RequestTimeout` | `RequestTimeout` | `Timeout` | `"Request timeout after 30 seconds"` |
| **Service unavailable** | `ServiceUnavailable` | `ServiceUnavailable` | `ServerBusy` | `"Service temporarily unavailable"` |

### Error Recovery Patterns

```python
def robust_upload(s3, bucket, key, content, max_attempts=3):
    """Upload with automatic retry and error recovery"""
    
    attempt = 1
    while attempt <= max_attempts:
        try:
            # Ensure bucket exists
            if not s3.bucket_exists(bucket):
                print("Creating bucket: {}".format(bucket))
                s3.create_bucket(bucket)
            
            # Attempt upload
            s3.put_object(bucket, key, content)
            print("Upload successful on attempt {}".format(attempt))
            return
            
        except Exception as e:
            error_message = str(e)
            
            # Determine if error is retryable
            retryable_errors = [
                "timeout",
                "service unavailable", 
                "internal server error",
                "network",
                "connection"
            ]
            
            is_retryable = any(retry_error in error_message.lower() 
                              for retry_error in retryable_errors)
            
            if not is_retryable or attempt == max_attempts:
                fail("Upload failed after {} attempts: {}".format(attempt, e))
            
            print("Attempt {} failed ({}), retrying...".format(attempt, e))
            attempt = attempt + 1
            
            # Exponential backoff
            import time
            time.sleep(min(2 ** (attempt - 1), 30))  # Cap at 30 seconds
    
    fail("Upload failed after {} attempts".format(max_attempts))
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

## Installation

```go
go get github.com/1set/starpkg/s3
```

## Quick Start Example

```python
load("s3", "connect")

# Create a client
s3 = connect(
    aws_access_key="YOUR_ACCESS_KEY",
    aws_secret_key="YOUR_SECRET_KEY",
    aws_region="us-east-1"
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

## Best Practices

### 1. **Security First**

#### Use Environment Variables for Credentials

```python
# Let the client read from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
s3 = connect()
```

#### Avoid Hardcoding Credentials

```python
# ❌ Bad: Hardcoded credentials
s3 = create_client(
    aws_access_key="AKIAIOSFODNN7EXAMPLE",
    aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

# ✅ Good: Environment-based credentials
s3 = create_client()  # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
```

#### Use Least Privilege Access

```python
# Create service-specific clients with limited permissions
backup_s3 = create_client(aws_region="us-east-1")  # Backup service account
web_s3 = create_client(aws_region="us-west-2")     # Web assets account
```

### 2. **Reliability Patterns**

#### Always Check Bucket Existence

```python
def ensure_bucket(s3, bucket_name):
    """Ensure bucket exists before operations"""
    if not s3.bucket_exists(bucket_name):
        try:
            s3.create_bucket(bucket_name)
            print("Created bucket: {}".format(bucket_name))
        except Exception as e:
            # Bucket might have been created by another process
            if "already exists" not in str(e).lower():
                fail("Failed to create bucket: {}".format(e))
```

#### Implement Retry Logic

```python
def retry_operation(operation, max_attempts=3):
    """Generic retry wrapper for S3 operations"""
    for attempt in range(1, max_attempts + 1):
        try:
            return operation()
        except Exception as e:
            if attempt == max_attempts:
                fail("Operation failed after {} attempts: {}".format(max_attempts, e))
            print("Attempt {} failed: {}".format(attempt, e))
            time.sleep(2 ** attempt)  # Exponential backoff
```

### 3. **Performance Optimization**

#### Use Appropriate Content Types

```python
# Define content type mapping
content_types = {
    ".html": "text/html",
    ".css": "text/css", 
    ".js": "application/javascript",
    ".json": "application/json",
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".pdf": "application/pdf"
}

def get_content_type(file_path):
    """Get content type from file extension"""
    ext = file_path.split('.')[-1].lower()
    return content_types.get('.' + ext, "application/octet-stream")

# Upload with proper content type
s3.put_object(
    "web-bucket",
    "index.html", 
    html_content,
    content_type=get_content_type("index.html")
)
```

#### Handle Large Files with Multi-part Upload

```python
def smart_upload(s3, bucket, key, content):
    """Choose upload method based on content size"""
    content_size = len(content)
    
    # Use multipart for files larger than 100MB
    if content_size > 100 * 1024 * 1024:
        return multipart_upload(s3, bucket, key, content)
    else:
        return s3.put_object(bucket, key, content)
```

#### Set Appropriate Timeouts

```python
# For large file operations
large_file_s3 = create_client(
    timeout=300,      # 5 minutes
    max_retries=5,    # More retries for large files
    part_size=10*1024*1024  # 10MB parts for faster uploads
)

# For quick metadata operations
quick_s3 = create_client(
    timeout=10,       # 10 seconds
    max_retries=2     # Fewer retries for quick operations
)
```

### 4. **Service-Specific Optimizations**

#### AWS S3 Optimizations

```python
# Use appropriate regions for performance
us_east_s3 = create_client(aws_region="us-east-1")  # Lowest latency for US East
eu_west_s3 = create_client(aws_region="eu-west-1")  # EU operations

# Enable request compression for text content
s3 = create_client(enable_compression=True)
```

#### MinIO Optimizations

```python
# MinIO requires path-style addressing
minio_s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    force_path_style=True,  # Required for MinIO
    use_ssl=False,          # For local development
    timeout=60              # Longer timeout for self-hosted
)
```

#### DigitalOcean Spaces Optimizations

```python
# Use CDN-friendly configurations
do_s3 = create_client(
    service_type="digitalocean",
    endpoint="https://nyc3.digitaloceanspaces.com",
    aws_region="nyc3"
)

# Set cache headers for CDN
s3.put_object(
    "cdn-bucket",
    "static/image.jpg",
    image_data,
    metadata={"Cache-Control": "public, max-age=31536000"}  # 1 year
)
```

### 5. **Resource Management**

#### Batch Operations for Efficiency

```python
def batch_delete(s3, bucket, keys):
    """Delete objects in batches for efficiency"""
    batch_size = 100  # Most services support up to 1000
    
    for i in range(0, len(keys), batch_size):
        batch = keys[i:i + batch_size]
        result = s3.delete_objects(bucket, batch)
        
        if "errors" in result:
            for error in result["errors"]:
                print("Failed to delete {}: {}".format(error["key"], error["message"]))
```

#### Lifecycle Management

```python
def cleanup_old_objects(s3, bucket, prefix, days_old=30):
    """Clean up objects older than specified days"""
    cutoff_date = time.now().add(-days_old * 24 * time.hour)
    
    objects = s3.list_objects(bucket, prefix=prefix)
    old_objects = []
    
    for obj in objects["contents"]:
        if obj["last_modified"] < cutoff_date:
            old_objects.append(obj["key"])
    
    if old_objects:
        print("Cleaning up {} old objects".format(len(old_objects)))
        s3.delete_objects(bucket, old_objects)
```

### 6. **Error Handling Best Practices**

#### Graceful Degradation

```python
def safe_get_object(s3, bucket, key, fallback=None):
    """Get object with fallback handling"""
    try:
        return s3.get_object(bucket, key)
    except Exception as e:
        if "not found" in str(e).lower():
            print("Object not found: s3://{}/{}, using fallback".format(bucket, key))
            return fallback
        else:
            fail("Failed to get object: {}".format(e))
```

#### Detailed Error Reporting

```python
def detailed_error_handling(s3, bucket, key):
    """Provide detailed error information"""
    try:
        return s3.get_object(bucket, key)
    except Exception as e:
        error_msg = str(e)
        
        # Add context to error message
        context = "Operation: get_object, Bucket: {}, Key: {}".format(bucket, key)
        
        if "not found" in error_msg.lower():
            fail("{} - Object does not exist".format(context))
        elif "access denied" in error_msg.lower():
            fail("{} - Permission denied (check credentials and policies)".format(context))
        elif "timeout" in error_msg.lower():
            fail("{} - Request timeout (check network connectivity)".format(context))
        else:
            fail("{} - Unexpected error: {}".format(context, error_msg))
```

### 7. **Migration Guidelines**

#### From AWS CLI to Starlark S3

```python
# AWS CLI: aws s3 cp file.txt s3://bucket/path/
# Starlark equivalent:
load("file", "read")
content = read("file.txt")
s3.put_object("bucket", "path/file.txt", content)

# AWS CLI: aws s3 sync ./local-dir s3://bucket/path/
# Starlark equivalent:
def sync_directory(s3, bucket, local_dir, s3_prefix):
    """Sync local directory to S3"""
    # Implementation would scan local directory
    # and upload changed files
```

#### From boto3 to Starlark S3

```python
# boto3: s3.create_bucket(Bucket='bucket-name')
# Starlark: s3.create_bucket('bucket-name')

# boto3: s3.put_object(Bucket='bucket', Key='key', Body=data)
# Starlark: s3.put_object('bucket', 'key', data)

# boto3: response = s3.list_objects_v2(Bucket='bucket')
# Starlark: objects = s3.list_objects('bucket')
```

### 8. **Testing and Validation**

#### Mock Testing with MinIO

```python
def setup_test_environment():
    """Set up MinIO for testing"""
    return connect(
        service_type="minio",
        endpoint="http://localhost:9000",
        aws_access_key="testkey",
        aws_secret_key="testsecret",
        force_path_style=True,
        use_ssl=False
    )

def test_bucket_operations():
    """Test basic bucket operations"""
    s3 = setup_test_environment()
    test_bucket = "test-bucket-{}".format(int(time.now().unix()))
    
    # Test create
    s3.create_bucket(test_bucket)
    
    # Test exists
    if not s3.bucket_exists(test_bucket):
        fail("Bucket should exist after creation")
    
    # Test delete
    s3.delete_bucket(test_bucket)
    
    print("Bucket operations test passed")
```

#### Validation Helpers

```python
def validate_s3_config(s3):
    """Validate S3 client configuration"""
    try:
        # Test basic connectivity
        buckets = s3.list_buckets()
        print("S3 connectivity verified - {} buckets accessible".format(len(buckets)))
        return True
    except Exception as e:
        print("S3 configuration validation failed: {}".format(e))
        return False
```

## Troubleshooting Guide

### Common Issues and Solutions

#### **Authentication Problems**

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Invalid credentials** | `Authentication failed: invalid credentials` | Verify `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables |
| **Expired STS tokens** | `Token has expired` | Refresh temporary credentials or use long-term access keys |
| **Region mismatch** | `SignatureDoesNotMatch` | Ensure client region matches bucket region |
| **Wrong endpoint** | `Connection refused` or `DNS resolution failed` | Verify endpoint URL and network connectivity |

#### **Network and Connectivity Issues**

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Timeout errors** | `Request timeout after 30 seconds` | Increase timeout value or check network connectivity |
| **SSL/TLS errors** | `SSL certificate verification failed` | Set `use_ssl=False` for testing or fix certificate issues |
| **Proxy configuration** | `Connection refused` | Configure HTTP_PROXY/HTTPS_PROXY environment variables |
| **DNS resolution** | `No such host` | Verify endpoint URL or use IP address |

#### **Service-Specific Issues**

| Service | Issue | Solution |
|---------|-------|----------|
| **MinIO** | `Path-style addressing required` | Set `force_path_style=True` |
| **DigitalOcean Spaces** | `Region not supported` | Use correct regional endpoint |
| **Azure Blob Storage** | `Limited S3 API support` | Check feature compatibility table |
| **Backblaze B2** | `Pre-signed URL failures` | Use alternative download methods |

#### **Performance Issues**

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Slow uploads** | `Upload taking too long` | Use multipart upload, increase `part_size`, or add concurrency |
| **Memory usage** | `Out of memory errors` | Use streaming uploads for large files |
| **Rate limiting** | `Too many requests` | Implement exponential backoff retry logic |
| **Large file timeouts** | `Timeout on large files` | Increase timeout and use multipart upload |

### Diagnostic Commands

#### Test Connectivity

```python
def diagnose_connection(s3):
    """Diagnose S3 connection issues"""
    try:
        buckets = s3.list_buckets()
        print("✅ Connection successful - {} buckets found".format(len(buckets)))
        return True
    except Exception as e:
        error_msg = str(e).lower()
        
        if "authentication" in error_msg or "access denied" in error_msg:
            print("❌ Authentication failed - check credentials")
        elif "timeout" in error_msg or "connection" in error_msg:
            print("❌ Network connectivity issue - check endpoint and firewall")
        elif "signature" in error_msg:
            print("❌ Signature mismatch - check region and endpoint configuration")
        else:
            print("❌ Unexpected error: {}".format(e))
        
        return False
```

#### Test Upload Performance

```python
def test_upload_performance(s3, bucket):
    """Test upload performance with different configurations"""
    test_data = "x" * (1024 * 1024)  # 1MB test data
    test_key = "performance-test-{}".format(int(time.now().unix()))
    
    start_time = time.now()
    s3.put_object(bucket, test_key, test_data)
    duration = time.now() - start_time
    
    print("Upload performance: {:.2f} seconds for 1MB".format(duration))
    
    # Cleanup
    s3.delete_object(bucket, test_key)
```

### Environment Setup Validation

#### Required Environment Variables

```bash
# Minimum required for AWS S3
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"

# Optional but recommended
export S3_TIMEOUT="30"
export S3_MAX_RETRIES="3"
export S3_ENABLE_LOGGING="false"
```

#### Validation Script

```python
def validate_environment():
    """Validate environment configuration"""
    required_vars = ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]
    missing_vars = []
    
    for var in required_vars:
        if runtime.getenv(var) == None or runtime.getenv(var) == "":
            missing_vars.append(var)
    
    if missing_vars:
        fail("Missing required environment variables: {}".format(", ".join(missing_vars)))
    
    print("✅ Environment validation passed")
```

## Implementation Checklist

### Phase 1: Foundation (Week 1)

- [ ] Set up Go module structure with base package integration
- [ ] Implement configuration system with all required options
- [ ] Create S3 client wrapper with connection management
- [ ] Add basic bucket operations (create, delete, list, exists)
- [ ] Implement core object operations (put, get, delete)
- [ ] Add comprehensive error handling and normalization
- [ ] Write unit tests for configuration and basic operations
- [ ] Create MinIO integration test setup

### Phase 2: Object Management (Week 2)

- [ ] Implement advanced object operations (copy, move, list with filters)
- [ ] Add file upload/download with streaming support
- [ ] Create object metadata and property management
- [ ] Add input validation for bucket names and object keys
- [ ] Implement batch operations for multiple objects
- [ ] Add object existence checking and info retrieval
- [ ] Write integration tests for all object operations
- [ ] Add performance benchmarks for large files

### Phase 3: Advanced Features (Week 3)

- [ ] Implement multipart upload for large files
- [ ] Add pre-signed URL generation (GET, PUT, POST)
- [ ] Create object tagging and metadata management
- [ ] Add server-side encryption options
- [ ] Implement lifecycle management helpers
- [ ] Add concurrent upload/download support
- [ ] Write tests for advanced features
- [ ] Performance optimization and memory management

### Phase 4: Multi-Service Support (Week 4)

- [ ] Test and validate with AWS S3
- [ ] Implement MinIO-specific optimizations
- [ ] Add DigitalOcean Spaces configuration
- [ ] Test Azure Blob Storage compatibility
- [ ] Validate Backblaze B2 integration
- [ ] Add service-specific error handling
- [ ] Create service compatibility documentation
- [ ] Write cross-service integration tests

### Phase 5: Documentation and Polish (Week 5)

- [ ] Complete comprehensive documentation
- [ ] Add all usage examples and best practices
- [ ] Create migration guides from other tools
- [ ] Add troubleshooting and diagnostic tools
- [ ] Performance benchmarking and optimization
- [ ] Security review and validation
- [ ] Final integration testing
- [ ] Release preparation and versioning

### Quality Gates

Each phase must meet these criteria before proceeding:

#### **Code Quality**

- [ ] All unit tests pass with >90% coverage
- [ ] Integration tests pass with real services
- [ ] Code follows Go best practices and conventions
- [ ] Error handling is comprehensive and consistent
- [ ] Memory usage is optimized for large files

#### **Documentation Quality**

- [ ] All public functions have comprehensive documentation
- [ ] Examples are tested and working
- [ ] Error messages are clear and actionable
- [ ] Migration paths are documented
- [ ] Performance characteristics are documented

#### **Compatibility**

- [ ] Works with all major S3-compatible services
- [ ] Handles service-specific quirks gracefully
- [ ] Maintains consistent API across services
- [ ] Error messages are normalized across services
- [ ] Configuration is flexible and intuitive

### Dependencies and Resources

#### **Go Dependencies**

```go
// Core AWS SDK
github.com/aws/aws-sdk-go-v2/service/s3
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/credentials

// Base package for configuration
github.com/1set/starpkg/base

// Standard library
sync
time
fmt
errors
```

#### **External Resources**

- [AWS S3 API Reference](https://docs.aws.amazon.com/s3/latest/API/)
- [MinIO Documentation](https://docs.min.io/)
- [DigitalOcean Spaces API](https://docs.digitalocean.com/products/spaces/reference/s3-sdk-examples/)
- [Azure Blob Storage S3 API](https://docs.microsoft.com/en-us/azure/storage/blobs/storage-blob-s3-api)

#### **Testing Resources**

- MinIO server for local testing
- AWS S3 sandbox environment
- DigitalOcean Spaces test account
- Azure Blob Storage test account

### Success Metrics

#### **Performance Targets**

- [ ] 1000+ operations/second for metadata operations
- [ ] <100ms latency for small object operations
- [ ] 100MB/s+ throughput for large file uploads
- [ ] <50MB memory usage for typical workloads
- [ ] Stream 1GB+ files without memory issues

#### **Reliability Targets**

- [ ] 99.9% operation success rate
- [ ] Automatic retry with exponential backoff
- [ ] Graceful handling of network interruptions
- [ ] Data integrity verification with checksums
- [ ] Clear error messages for all failure scenarios

#### **Usability Targets**

- [ ] Intuitive API matching Starlark conventions
- [ ] Comprehensive examples for all features
- [ ] Easy migration from existing tools
- [ ] Consistent behavior across all services
- [ ] Minimal configuration required for common use cases

This comprehensive plan provides a solid foundation for implementing a production-ready S3 module for Starlark that follows best practices and integrates seamlessly with the existing ecosystem.
