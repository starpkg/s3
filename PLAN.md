# S3 Starlark Module Development Plan

## 🗂️ S3 Module - Simple Storage Service Operations for Starlark

**Module Name**: `s3`  
**Emoji**: 🗂️  
**Description**: Complete S3-compatible storage operations for Starlark  
**Tagline**: Unified interface for Amazon S3, MinIO, and all S3-compatible storage services

## Key Features

- 🔐 **Multiple Authentication Methods** - Support for access keys, environment variables, and IAM roles
- 🪣 **Comprehensive Bucket Operations** - Create, delete, list, and manage bucket configurations
- 📁 **Full Object Management** - Upload, download, copy, move, and delete objects with ease
- 🏷️ **Metadata & Tagging** - Handle custom metadata and object tags
- 🔗 **Pre-signed URLs** - Generate temporary access links for private objects
- 📦 **Multi-part Uploads** - Efficiently handle large file uploads
- 🌍 **Multi-Service Support** - Works with Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, and MinIO
- ⚡ **High Performance** - Optimized for speed with streaming and concurrent operations

## Executive Summary

The `s3` module provides comprehensive S3-compatible storage operations for Starlark scripts. It focuses on simplicity, security, and performance while supporting all major S3-compatible services including Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, and MinIO. The design emphasizes ease of use with powerful features for both simple scripts and complex applications.

## Supported S3-Compatible Services

### Service Support Matrix

| Service | Support Level | Configuration Required | Primary Use Case |
|---------|---------------|----------------------|------------------|
| **Amazon S3** | ✅ Complete | Default settings | Production cloud storage |
| **Cloudflare R2** | ✅ Complete | Custom endpoint | Edge storage, zero egress fees |
| **Backblaze B2** | ✅ Complete | S3-compatible API endpoint | Cost-effective backup |
| **DigitalOcean Spaces** | ✅ Complete | Custom endpoint | Developer-friendly hosting |
| **MinIO** | ✅ Complete | `force_path_style=True` | Self-hosted/on-premises |

**Legend:**
- ✅ **Complete**: Full API compatibility, all features supported

### Service-Specific Configuration

#### Amazon S3 (Reference Implementation)
```python
s3 = create_client(
    service_type="aws_s3",
    aws_region="us-east-1"
    # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
)
```

#### Cloudflare R2
```python
s3 = create_client(
    service_type="cloudflare_r2",
    endpoint="https://<account-id>.r2.cloudflarestorage.com",
    aws_access_key="YOUR_R2_ACCESS_KEY",
    aws_secret_key="YOUR_R2_SECRET_KEY",
    aws_region="auto"
)
```

#### Backblaze B2
```python
s3 = create_client(
    service_type="backblaze_b2",
    endpoint="https://s3.us-west-004.backblazeb2.com",
    aws_access_key="YOUR_APPLICATION_KEY_ID",
    aws_secret_key="YOUR_APPLICATION_KEY",
    aws_region="us-west-004"
)
```

#### DigitalOcean Spaces
```python
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
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    aws_access_key="minioadmin",
    aws_secret_key="minioadmin",
    aws_region="us-east-1",
    force_path_style=True,  # Required for MinIO
    use_ssl=False
)
```

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

#### `create_client()`
**Purpose**: Creates and configures an S3 client for interacting with S3-compatible storage services.

**Parameters**:
- `service_type` (string, optional): Service type - one of "aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean", "minio", or "auto". Default: "auto"
- `endpoint` (string, optional): Custom endpoint URL for S3-compatible services. Default: Auto-detected based on service_type
- `aws_region` (string, optional): AWS region or service region. Default: "us-east-1" or from environment `AWS_DEFAULT_REGION`
- `aws_access_key` (string, optional): AWS access key ID. Default: from environment `AWS_ACCESS_KEY_ID`
- `aws_secret_key` (string, optional): AWS secret access key. Default: from environment `AWS_SECRET_ACCESS_KEY`
- `aws_session_token` (string, optional): AWS session token for temporary credentials. Default: from environment `AWS_SESSION_TOKEN`
- `force_path_style` (bool, optional): Use path-style addressing (required for MinIO). Default: false
- `use_ssl` (bool, optional): Enable SSL/TLS encryption. Default: true
- `timeout` (int, optional): Connection timeout in seconds. Default: 30
- `max_retries` (int, optional): Maximum retry attempts for failed requests. Default: 3

**Returns**: S3Client object with methods for bucket and object operations

**Examples**:
```python
load("s3", "create_client")

# Create AWS S3 client with default settings
s3 = create_client()

# Create client with explicit configuration
s3 = create_client(
    service_type="aws_s3",
    aws_region="eu-west-1",
    timeout=60
)
```

#### `parse_s3_url(url)`
**Purpose**: Parses an S3 URL into bucket and key components.

**Parameters**:
- `url` (string): S3 URL in format "s3://bucket/key" or "s3://bucket/path/to/key"

**Returns**: Dictionary with "bucket" and "key" fields

**Example**:
```python
result = parse_s3_url("s3://my-bucket/path/to/file.txt")
print(result["bucket"])  # "my-bucket"
print(result["key"])     # "path/to/file.txt"
```

#### `generate_s3_url(bucket, key, region="us-east-1")`
**Purpose**: Generates an S3 URL from bucket, key, and region components.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path
- `region` (string, optional): AWS region. Default: "us-east-1"

**Returns**: String containing the S3 URL

**Example**:
```python
url = generate_s3_url("my-bucket", "path/to/file.txt", "eu-west-1")
print(url)  # "s3://my-bucket/path/to/file.txt"
```

#### `validate_bucket_name(name)`
**Purpose**: Validates bucket name according to S3 naming rules.

**Parameters**:
- `name` (string): Bucket name to validate

**Returns**: Boolean indicating if the name is valid

**Example**:
```python
if validate_bucket_name("my-bucket-123"):
    print("Valid bucket name")
```

#### `validate_object_key(key)`
**Purpose**: Validates object key according to S3 naming rules.

**Parameters**:
- `key` (string): Object key to validate

**Returns**: Boolean indicating if the key is valid

**Example**:
```python
if validate_object_key("path/to/file.txt"):
    print("Valid object key")
```

#### `get_supported_services()`
**Purpose**: Returns list of supported S3-compatible services.

**Parameters**: None

**Returns**: List of strings containing supported service types

**Example**:
```python
services = get_supported_services()
print(services)  # ["aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean", "minio"]
```

#### `get_client_info(client)`
**Purpose**: Returns information about the S3 client configuration.

**Parameters**:
- `client` (S3Client): S3 client object

**Returns**: Dictionary with client configuration details

**Example**:
```python
info = get_client_info(s3)
print(info["service_type"])  # "aws_s3"
print(info["region"])        # "us-east-1"
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

## S3Client Object API

### Bucket Operations

#### `create_bucket(name, region=None, **options)`
**Purpose**: Creates a new bucket in the specified region.

**Parameters**:
- `name` (string): Bucket name (must be globally unique for AWS S3)
- `region` (string, optional): Region to create bucket in. Default: client's default region
- `**options`: Additional bucket creation options (service-specific)

**Returns**: None

**Raises**: Error if bucket creation fails or bucket already exists

**Example**:
```python
s3.create_bucket("my-new-bucket")
s3.create_bucket("eu-bucket", region="eu-west-1")
```

#### `delete_bucket(name, force=False)`
**Purpose**: Deletes an existing bucket.

**Parameters**:
- `name` (string): Bucket name to delete
- `force` (bool, optional): If true, delete all objects in bucket first. Default: false

**Returns**: None

**Raises**: Error if bucket doesn't exist or contains objects (when force=false)

**Example**:
```python
s3.delete_bucket("empty-bucket")
s3.delete_bucket("full-bucket", force=True)  # Delete non-empty bucket
```

#### `list_buckets()`
**Purpose**: Lists all buckets accessible to the authenticated user.

**Parameters**: None

**Returns**: List of dictionaries, each containing bucket information:
- `name` (string): Bucket name
- `creation_date` (string): ISO 8601 timestamp of bucket creation
- `region` (string, optional): Bucket region (if available)

**Example**:
```python
buckets = s3.list_buckets()
for bucket in buckets:
    print("Bucket: {}, Created: {}".format(bucket["name"], bucket["creation_date"]))
```

#### `bucket_exists(name)`
**Purpose**: Checks if a bucket exists and is accessible.

**Parameters**:
- `name` (string): Bucket name to check

**Returns**: Boolean indicating if bucket exists

**Example**:
```python
if s3.bucket_exists("my-bucket"):
    print("Bucket exists!")
```

#### `get_bucket_location(name)`
**Purpose**: Gets the region/location of a bucket.

**Parameters**:
- `name` (string): Bucket name

**Returns**: String containing the bucket's region

**Raises**: Error if bucket doesn't exist or is not accessible

**Example**:
```python
location = s3.get_bucket_location("my-bucket")
print("Bucket region:", location)
```

#### `set_bucket_versioning(name, enabled=True)`
**Purpose**: Enables or disables versioning for a bucket.

**Parameters**:
- `name` (string): Bucket name
- `enabled` (bool, optional): Whether to enable versioning. Default: true

**Returns**: None

**Raises**: Error if bucket doesn't exist or versioning cannot be configured

**Example**:
```python
s3.set_bucket_versioning("my-bucket", enabled=True)
```

#### `get_bucket_versioning(name)`
**Purpose**: Gets the versioning configuration for a bucket.

**Parameters**:
- `name` (string): Bucket name

**Returns**: Dictionary containing versioning information:
- `enabled` (bool): Whether versioning is enabled
- `status` (string): Versioning status ("Enabled", "Suspended", or "Disabled")

**Example**:
```python
versioning = s3.get_bucket_versioning("my-bucket")
print("Versioning enabled:", versioning["enabled"])
```

### Object Operations - Core

#### `put_object(bucket, key, content, **options)`
**Purpose**: Uploads an object to the specified bucket.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path
- `content` (string or bytes): Object content to upload
- `**options`: Additional upload options:
  - `content_type` (string): MIME type of the content
  - `metadata` (dict): Custom metadata key-value pairs
  - `tags` (dict): Object tags key-value pairs
  - `encryption` (string): Server-side encryption type

**Returns**: Dictionary containing upload result:
- `etag` (string): ETag of the uploaded object
- `version_id` (string, optional): Version ID if versioning is enabled

**Example**:
```python
result = s3.put_object(
    "my-bucket", 
    "hello.txt", 
    "Hello, World!",
    content_type="text/plain",
    metadata={"author": "John Doe"}
)
print("Upload ETag:", result["etag"])
```

#### `put_object_from_file(bucket, key, file_path, **options)`
**Purpose**: Uploads a file to the specified bucket.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path
- `file_path` (string): Local file path to upload
- `**options`: Same options as `put_object()`

**Returns**: Dictionary containing upload result (same as `put_object()`)

**Raises**: Error if file doesn't exist or cannot be read

**Example**:
```python
s3.put_object_from_file("my-bucket", "image.jpg", "/path/to/image.jpg")
```

#### `get_object(bucket, key)`
**Purpose**: Downloads an object's content to memory.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path

**Returns**: String or bytes containing the object content

**Raises**: Error if object doesn't exist or cannot be accessed

**Example**:
```python
content = s3.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"
```

#### `get_object_to_file(bucket, key, file_path)`
**Purpose**: Downloads an object directly to a local file.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path
- `file_path` (string): Local file path to save to

**Returns**: None

**Raises**: Error if object doesn't exist or file cannot be written

**Example**:
```python
s3.get_object_to_file("my-bucket", "image.jpg", "/local/path/image.jpg")
```

#### `delete_object(bucket, key)`
**Purpose**: Deletes a single object from the bucket.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key/path

**Returns**: None

**Raises**: Error if deletion fails (object not existing is not an error)

**Example**:
```python
s3.delete_object("my-bucket", "hello.txt")
```

#### `delete_objects(bucket, keys)`
**Purpose**: Deletes multiple objects from the bucket in a single request.

**Parameters**:
- `bucket` (string): Bucket name
- `keys` (list): List of object keys/paths to delete

**Returns**: Dictionary containing deletion results:
- `deleted` (list): List of successfully deleted object keys
- `errors` (list, optional): List of deletion errors (if any)

**Example**:
```python
result = s3.delete_objects("my-bucket", ["file1.txt", "file2.txt", "file3.txt"])
print("Deleted:", len(result["deleted"]))
if "errors" in result:
    print("Errors:", result["errors"])
```

### Object Operations - Advanced

#### `copy_object(src_bucket, src_key, dst_bucket, dst_key, **options)`
**Purpose**: Copies an object from one location to another.

**Parameters**:
- `src_bucket` (string): Source bucket name
- `src_key` (string): Source object key
- `dst_bucket` (string): Destination bucket name
- `dst_key` (string): Destination object key
- `**options`: Copy options:
  - `metadata_directive` (string): "COPY" or "REPLACE"
  - `metadata` (dict): New metadata (if REPLACE)
  - `tags` (dict): New tags

**Returns**: Dictionary containing copy result:
- `etag` (string): ETag of the copied object
- `copy_source_version_id` (string, optional): Source version ID

**Example**:
```python
s3.copy_object("source-bucket", "source.txt", "dest-bucket", "dest.txt")
```

#### `move_object(src_bucket, src_key, dst_bucket, dst_key, **options)`
**Purpose**: Moves an object from one location to another (copy + delete).

**Parameters**:
- Same as `copy_object()`

**Returns**: Dictionary containing move result (same as `copy_object()`)

**Example**:
```python
s3.move_object("old-bucket", "file.txt", "new-bucket", "file.txt")
```

#### `list_objects(bucket, prefix="", delimiter="", max_keys=1000)`
**Purpose**: Lists objects in a bucket with optional filtering.

**Parameters**:
- `bucket` (string): Bucket name
- `prefix` (string, optional): Object key prefix filter
- `delimiter` (string, optional): Delimiter for hierarchical listing
- `max_keys` (int, optional): Maximum number of keys to return. Default: 1000

**Returns**: Dictionary containing listing results:
- `contents` (list): List of object information dictionaries:
  - `key` (string): Object key
  - `size` (int): Object size in bytes
  - `last_modified` (string): ISO 8601 timestamp
  - `etag` (string): Object ETag
  - `storage_class` (string): Storage class
- `common_prefixes` (list, optional): Common prefixes (if delimiter used)
- `is_truncated` (bool): Whether more results are available
- `next_marker` (string, optional): Marker for next page

**Example**:
```python
objects = s3.list_objects("my-bucket", prefix="photos/2024/")
for obj in objects["contents"]:
    print("{} ({} bytes)".format(obj["key"], obj["size"]))
```

#### `get_object_info(bucket, key)`
**Purpose**: Gets metadata and properties of an object without downloading content.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary containing object information:
- `size` (int): Object size in bytes
- `last_modified` (string): ISO 8601 timestamp
- `etag` (string): Object ETag
- `content_type` (string): MIME type
- `metadata` (dict): Custom metadata
- `version_id` (string, optional): Version ID if versioning enabled

**Example**:
```python
info = s3.get_object_info("my-bucket", "hello.txt")
print("Size: {} bytes".format(info["size"]))
print("Last modified: {}".format(info["last_modified"]))
```

#### `object_exists(bucket, key)`
**Purpose**: Checks if an object exists in the bucket.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Boolean indicating if object exists

**Example**:
```python
if s3.object_exists("my-bucket", "hello.txt"):
    print("Object exists!")
```

### Metadata and Tagging

#### `get_object_metadata(bucket, key)`
**Purpose**: Retrieves custom metadata for an object.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary containing custom metadata key-value pairs

**Example**:
```python
metadata = s3.get_object_metadata("my-bucket", "document.pdf")
print("Author:", metadata.get("author"))
```

#### `set_object_metadata(bucket, key, metadata)`
**Purpose**: Sets custom metadata for an object (requires copy operation).

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `metadata` (dict): Metadata key-value pairs to set

**Returns**: None

**Example**:
```python
s3.set_object_metadata("my-bucket", "file.txt", {
    "author": "Jane Doe",
    "version": "2.0"
})
```

#### `get_object_tags(bucket, key)`
**Purpose**: Retrieves tags for an object.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary containing tag key-value pairs

**Example**:
```python
tags = s3.get_object_tags("my-bucket", "report.pdf")
for key, value in tags.items():
    print("{}: {}".format(key, value))
```

#### `set_object_tags(bucket, key, tags)`
**Purpose**: Sets tags for an object.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `tags` (dict): Tag key-value pairs to set

**Returns**: None

**Example**:
```python
s3.set_object_tags("my-bucket", "report.pdf", {
    "environment": "production",
    "confidential": "true",
    "project": "alpha"
})
```

#### `delete_object_tags(bucket, key)`
**Purpose**: Removes all tags from an object.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: None

**Example**:
```python
s3.delete_object_tags("my-bucket", "report.pdf")
```

### Pre-signed URLs

#### `presign_url(bucket, key, expires_in=3600, method="GET")`
**Purpose**: Generates a pre-signed URL for object access.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration time in seconds. Default: 3600 (1 hour)
- `method` (string, optional): HTTP method ("GET", "PUT", "DELETE"). Default: "GET"

**Returns**: String containing the pre-signed URL

**Example**:
```python
url = s3.presign_url("my-bucket", "private.pdf", expires_in=3600)
print("Download URL:", url)
```

#### `presign_put_url(bucket, key, expires_in=3600, **options)`
**Purpose**: Generates a pre-signed URL for uploading an object.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration time in seconds. Default: 3600
- `**options`: Upload constraints:
  - `content_type` (string): Required content type
  - `content_length` (int): Required content length

**Returns**: String containing the pre-signed PUT URL

**Example**:
```python
url = s3.presign_put_url("my-bucket", "upload.jpg", content_type="image/jpeg")
```

#### `presign_post(bucket, key, expires_in=3600, **options)`
**Purpose**: Generates pre-signed POST data for browser uploads.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): Expiration time in seconds. Default: 3600
- `**options`: Upload constraints and conditions

**Returns**: Dictionary containing:
- `url` (string): POST URL
- `fields` (dict): Form fields to include in POST request

**Example**:
```python
post_data = s3.presign_post("my-bucket", "upload.jpg")
print("POST URL:", post_data["url"])
print("Form fields:", post_data["fields"])
```

### Multi-part Upload

#### `create_multipart_upload(bucket, key, **options)`
**Purpose**: Initiates a multi-part upload for large objects.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `**options`: Upload options (same as `put_object()`)

**Returns**: String containing the upload ID

**Example**:
```python
upload_id = s3.create_multipart_upload(
    "backup-bucket",
    "large-backup.tar.gz",
    content_type="application/gzip"
)
```

#### `upload_part(bucket, key, upload_id, part_number, content)`
**Purpose**: Uploads a single part in a multi-part upload.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Multi-part upload ID
- `part_number` (int): Part number (1-10000)
- `content` (string or bytes): Part content

**Returns**: Dictionary containing part information:
- `part_number` (int): Part number
- `etag` (string): Part ETag

**Example**:
```python
part = s3.upload_part("bucket", "large-file.zip", upload_id, 1, part_data)
```

#### `complete_multipart_upload(bucket, key, upload_id, parts)`
**Purpose**: Completes a multi-part upload by combining all parts.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Multi-part upload ID
- `parts` (list): List of part dictionaries from `upload_part()`

**Returns**: Dictionary containing completion result:
- `etag` (string): Final object ETag
- `location` (string): Object URL

**Example**:
```python
result = s3.complete_multipart_upload("bucket", "large-file.zip", upload_id, parts)
print("Upload completed. ETag:", result["etag"])
```

#### `abort_multipart_upload(bucket, key, upload_id)`
**Purpose**: Aborts an incomplete multi-part upload.

**Parameters**:
- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Multi-part upload ID

**Returns**: None

**Example**:
```python
s3.abort_multipart_upload("bucket", "large-file.zip", upload_id)
```

#### `list_multipart_uploads(bucket, prefix="")`
**Purpose**: Lists incomplete multi-part uploads for a bucket.

**Parameters**:
- `bucket` (string): Bucket name
- `prefix` (string, optional): Key prefix filter

**Returns**: List of dictionaries containing upload information:
- `upload_id` (string): Upload ID
- `key` (string): Object key
- `initiated` (string): ISO 8601 timestamp of initiation

**Example**:
```python
uploads = s3.list_multipart_uploads("backup-bucket")
for upload in uploads:
    print("Upload ID: {}, Key: {}".format(upload["upload_id"], upload["key"]))
```

## Usage Examples

### Basic File Management

```python
load("s3", "create_client")

def main():
    s3 = create_client(aws_region="us-east-1")
    
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

### Complex Examples

For more comprehensive examples, see the `examples/` directory:

- [`examples/website_deployment.star`](examples/website_deployment.star) - Deploy static website files to S3
- [`examples/backup_system.star`](examples/backup_system.star) - Automated backup system with lifecycle management
- [`examples/data_pipeline.star`](examples/data_pipeline.star) - Data processing pipeline between S3 buckets
- [`examples/multi_service.star`](examples/multi_service.star) - Working with multiple S3-compatible services
- [`examples/error_handling.star`](examples/error_handling.star) - Robust error handling and validation patterns

## Feature Compatibility by Service

### Core Operations Support

| Operation | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Notes |
|-----------|-----------|---------------|--------------|--------------|-------|-------|
| **Bucket Operations** |
| `create_bucket()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_bucket()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `list_buckets()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `bucket_exists()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_bucket_location()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `set_bucket_versioning()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| `get_bucket_versioning()` | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| **Object Operations** |
| `put_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_object()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `delete_objects()` (batch) | ✅ | ✅ | ⚠️ | ✅ | ✅ | Limited batch size on B2 |
| `copy_object()` | ✅ | ✅ | ⚠️ | ✅ | ✅ | Some restrictions on B2 |
| `list_objects()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `get_object_info()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| `object_exists()` | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |

### Advanced Features Support

| Feature | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO | Limitations |
|---------|-----------|---------------|--------------|--------------|-------|-------------|
| **Multipart Upload** | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| **Pre-signed URLs** | ✅ | ✅ | ⚠️ | ✅ | ✅ | Limited on B2 |
| **Object Tagging** | ✅ | ❌ | ❌ | ❌ | ✅ | AWS S3 and MinIO only |
| **Custom Metadata** | ✅ | ✅ | ✅ | ✅ | ✅ | Universal support |
| **Server-side Encryption** | ✅ | ✅ | ✅ | ⚠️ | ✅ | Varies by service |

**Legend:**
- ✅ **Full Support**: All features work as expected
- ⚠️ **Limited Support**: Core functionality works with some restrictions
- ❌ **Not Supported**: Feature not available

### Authentication Methods by Service

| Authentication Method | Amazon S3 | Cloudflare R2 | Backblaze B2 | DigitalOcean | MinIO |
|----------------------|-----------|---------------|--------------|--------------|-------|
| **Access Key + Secret** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Environment Variables** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **IAM Roles** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **STS Tokens** | ✅ | ❌ | ❌ | ❌ | ❌ |

## Environment Variable Configuration

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

## Quick Start Example

```python
load("s3", "create_client")

# Create a client
s3 = create_client(
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
├── examples/       # Example Starlark scripts
│   ├── website_deployment.star
│   ├── backup_system.star
│   ├── data_pipeline.star
│   ├── multi_service.star
│   └── error_handling.star
├── README.md       # User documentation
├── go.mod
└── go.sum
```

## Development Plan

### Phase 1: Core Infrastructure (Week 1)
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
- [ ] Test and validate with Amazon S3
- [ ] Implement Cloudflare R2 configuration and testing
- [ ] Add Backblaze B2 configuration and testing
- [ ] Test DigitalOcean Spaces compatibility
- [ ] Validate MinIO integration
- [ ] Add service-specific error handling
- [ ] Create service compatibility documentation
- [ ] Write cross-service integration tests

### Phase 5: Documentation and Polish (Week 5)
- [ ] Complete comprehensive documentation with function details
- [ ] Create all usage examples and move complex ones to separate files
- [ ] Add migration guides from other tools
- [ ] Add troubleshooting and diagnostic tools
- [ ] Performance benchmarking and optimization
- [ ] Security review and validation
- [ ] Final integration testing
- [ ] Release preparation and versioning

## Success Metrics

### Performance Targets
- [ ] 1000+ operations/second for metadata operations
- [ ] <100ms latency for small object operations
- [ ] 100MB/s+ throughput for large file uploads
- [ ] <50MB memory usage for typical workloads
- [ ] Stream 1GB+ files without memory issues

### Reliability Targets
- [ ] 99.9% operation success rate
- [ ] Automatic retry with exponential backoff
- [ ] Graceful handling of network interruptions
- [ ] Data integrity verification with checksums
- [ ] Clear error messages for all failure scenarios

### Usability Targets
- [ ] Intuitive API matching Starlark conventions
- [ ] Comprehensive examples for all features
- [ ] Easy migration from existing tools
- [ ] Consistent behavior across all services
- [ ] Minimal configuration required for common use cases
