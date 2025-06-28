# S3 Starlark Module Development Plan

## 🗂️ S3 Module - Simple Storage Service Operations for Starlark

**Module Name**: `s3`  
**Emoji**: 🗂️  
**Description**: Complete S3-compatible storage operations for Starlark  
**Tagline**: Unified interface for Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, MinIO, and other S3-compatible services

## Key Features

- 🔐 **Multiple Authentication Methods** - Support for access keys, environment variables, and IAM roles
- 🪣 **Comprehensive Bucket Operations** - Create, delete, list, and manage bucket configurations
- 📁 **Full Object Management** - Upload, download, copy, move, and delete objects with ease
- 🏷️ **Metadata & Tagging** - Handle custom metadata and object tags
- 🔗 **Pre-signed URLs** - Generate temporary access links for private objects
- 📦 **Multi-part Uploads** - Efficiently handle large file uploads
- 🌍 **Multi-Service Support** - Works with Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, MinIO, and other S3-compatible services
- ⚡ **High Performance** - Optimized for speed with streaming and concurrent operations

## Executive Summary

The `s3` module provides comprehensive S3-compatible storage operations for Starlark scripts. It focuses on simplicity, security, and performance while supporting all major S3-compatible services including Amazon S3, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, and MinIO. The design emphasizes ease of use with powerful features for both simple scripts and complex applications.

## Quick Start

```python
load("s3", "create_client")

# Create client (uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment)
s3 = create_client()

# Create a bucket
s3.create_bucket("my-bucket")

# Upload a file
s3.put_object("my-bucket", "hello.txt", "Hello, World!")

# Download the file
content = s3.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"

# List objects
objects = s3.list_objects("my-bucket")
for obj in objects["contents"]:
    print("{} ({} bytes)".format(obj["key"], obj["size"]))
```

For other S3-compatible services:

```python
# MinIO
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    force_path_style=True,
    use_ssl=False
)

# Cloudflare R2
s3 = create_client(
    service_type="cloudflare_r2",
    endpoint="https://<account-id>.r2.cloudflarestorage.com",
    access_key="YOUR_R2_ACCESS_KEY",
    secret_key="YOUR_R2_SECRET_KEY"
)
```

## Supported S3-Compatible Services

The module provides first-class support for the following S3-compatible storage services:

### Service Support Matrix

| Service | Support Level | Configuration Notes | Primary Use Case |
|---------|---------------|-------------------|------------------|
| **Amazon S3** | ✅ Complete | Default configuration | Production cloud storage |
| **Cloudflare R2** | ✅ Complete | Custom endpoint required | Edge storage, cost-effective |
| **Backblaze B2** | ✅ Complete | S3-compatible API mode | Cost-effective backup and storage |
| **DigitalOcean Spaces** | ✅ Complete | Custom endpoint required | Developer-friendly hosting |
| **MinIO** | ✅ Complete | `force_path_style=True` required | Self-hosted/on-premises storage |

All services support the core S3 API features including bucket operations, object CRUD, metadata management, multipart uploads, and pre-signed URLs.

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

#### `create_client(service_type="auto", **config) -> S3Client`

**Purpose**: Creates and configures an S3-compatible client for interacting with storage services.

**Parameters**:

- `service_type` (string, optional): Target service type. Options: `"aws_s3"`, `"cloudflare_r2"`, `"backblaze_b2"`, `"digitalocean_spaces"`, `"minio"`, `"auto"`. Default: `"auto"`
- `endpoint` (string, optional): Custom endpoint URL for S3-compatible services. Auto-detected based on service_type if not provided
- `region` (string, optional): AWS region or equivalent. Default: `"us-east-1"` or `AWS_DEFAULT_REGION` environment variable
- `access_key` (string, optional): Access key ID. Default: `AWS_ACCESS_KEY_ID` environment variable
- `secret_key` (string, optional): Secret access key. Default: `AWS_SECRET_ACCESS_KEY` environment variable  
- `session_token` (string, optional): Session token for temporary credentials. Default: `AWS_SESSION_TOKEN` environment variable
- `force_path_style` (bool, optional): Use path-style addressing (required for MinIO). Default: `False`
- `use_ssl` (bool, optional): Enable/disable SSL. Default: `True`
- `timeout` (int, optional): Connection timeout in seconds. Default: `30`
- `max_retries` (int, optional): Maximum retry attempts. Default: `3`
- `part_size` (int, optional): Multi-part upload part size in bytes. Default: `5242880` (5MB)
- `concurrency` (int, optional): Concurrent uploads/downloads. Default: `3`
- `enable_logging` (bool, optional): Enable request logging. Default: `False`
- `user_agent` (string, optional): Custom user agent. Default: `"starlark-s3/1.0"`

**Returns**: `S3Client` object with methods for S3 operations

**Example**:

```python
s3 = create_client(
    service_type="aws_s3",
    region="us-east-1",
    timeout=60
)
```

#### `parse_s3_url(url) -> dict`

**Purpose**: Parses an S3 URL into bucket and key components.

**Parameters**:

- `url` (string): S3 URL in format `s3://bucket-name/object-key`

**Returns**: Dictionary with `"bucket"` and `"key"` fields

**Example**:

```python
parsed = parse_s3_url("s3://my-bucket/path/to/file.txt")
# Returns: {"bucket": "my-bucket", "key": "path/to/file.txt"}
```

#### `generate_s3_url(bucket, key, region="us-east-1") -> string`

**Purpose**: Generates a standard S3 URL for the given bucket and key.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `region` (string, optional): AWS region. Default: `"us-east-1"`

**Returns**: S3 URL string

**Example**:

```python
url = generate_s3_url("my-bucket", "file.txt", "eu-west-1")
# Returns: "s3://my-bucket/file.txt"
```

#### `validate_bucket_name(name) -> bool`

**Purpose**: Validates bucket name according to S3 naming rules.

**Parameters**:

- `name` (string): Bucket name to validate

**Returns**: `True` if valid, `False` otherwise

#### `validate_object_key(key) -> bool`

**Purpose**: Validates object key according to S3 naming rules.

**Parameters**:

- `key` (string): Object key to validate

**Returns**: `True` if valid, `False` otherwise

#### `get_supported_services() -> list`

**Purpose**: Returns list of supported S3-compatible service types.

**Returns**: List of service type strings

**Example**:

```python
services = get_supported_services()
# Returns: ["aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean_spaces", "minio"]
```

#### `get_client_info(client) -> dict`

**Purpose**: Returns connection details and configuration of an S3 client.

**Parameters**:

- `client` (S3Client): S3 client instance

**Returns**: Dictionary with client configuration details

### Client Creation Examples

```python
load("s3", "create_client")

# Create a client with credentials
s3 = create_client(
    service_type="aws_s3",
    access_key="YOUR_ACCESS_KEY",
    secret_key="YOUR_SECRET_KEY",
    region="us-east-1"
)

# Or use environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
s3 = create_client()  # Auto-detects AWS S3 with environment variables

# For Cloudflare R2
s3 = create_client(
    service_type="cloudflare_r2",
    endpoint="https://<account-id>.r2.cloudflarestorage.com",
    access_key="YOUR_R2_ACCESS_KEY",
    secret_key="YOUR_R2_SECRET_KEY",
    region="auto"
)

# For MinIO (self-hosted)
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    region="us-east-1",
    force_path_style=True,  # Required for MinIO
    use_ssl=False
)

# For DigitalOcean Spaces
s3 = create_client(
    service_type="digitalocean_spaces",
    endpoint="https://nyc3.digitaloceanspaces.com",
    access_key="YOUR_SPACES_KEY",
    secret_key="YOUR_SPACES_SECRET",
    region="nyc3"
)

# For Backblaze B2
s3 = create_client(
    service_type="backblaze_b2",
    endpoint="https://s3.us-west-004.backblazeb2.com",
    access_key="YOUR_APPLICATION_KEY_ID",
    secret_key="YOUR_APPLICATION_KEY",
    region="us-west-004"
)
```

## Configuration Options

The S3 module supports various configuration options:

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `service_type` | string | Service type ("aws_s3", "cloudflare_r2", "backblaze_b2", "digitalocean_spaces", "minio", "auto") | `auto` |
| `access_key` | string | Access key ID | Environment: `AWS_ACCESS_KEY_ID` |
| `secret_key` | string | Secret access key | Environment: `AWS_SECRET_ACCESS_KEY` |
| `session_token` | string | Session token | Environment: `AWS_SESSION_TOKEN` |
| `region` | string | Region | Environment: `AWS_DEFAULT_REGION` or `us-east-1` |
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

##### `create_bucket(name, region=None, **options) -> None`

**Purpose**: Creates a new bucket in the specified region.

**Parameters**:

- `name` (string): Bucket name (must be globally unique and follow S3 naming rules)
- `region` (string, optional): Region to create bucket in. Uses client's default region if not specified
- `**options`: Additional bucket creation options (service-specific)

**Raises**: Error if bucket already exists or name is invalid

##### `delete_bucket(name, force=False) -> None`

**Purpose**: Deletes a bucket. Can optionally force delete non-empty buckets.

**Parameters**:

- `name` (string): Bucket name to delete
- `force` (bool, optional): If `True`, deletes all objects in bucket first. Default: `False`

**Raises**: Error if bucket doesn't exist or contains objects (when force=False)

##### `list_buckets() -> list`

**Purpose**: Lists all buckets accessible to the client.

**Returns**: List of dictionaries with bucket information (`name`, `creation_date`, `region`)

##### `bucket_exists(name) -> bool`

**Purpose**: Checks if a bucket exists and is accessible.

**Parameters**:

- `name` (string): Bucket name to check

**Returns**: `True` if bucket exists, `False` otherwise

##### `get_bucket_location(name) -> string`

**Purpose**: Gets the region/location of a bucket.

**Parameters**:

- `name` (string): Bucket name

**Returns**: Region string (e.g., "us-east-1", "eu-west-1")

##### `set_bucket_versioning(name, enabled=True) -> None`

**Purpose**: Enables or disables versioning for a bucket.

**Parameters**:

- `name` (string): Bucket name
- `enabled` (bool, optional): Enable versioning if `True`. Default: `True`

##### `get_bucket_versioning(name) -> dict`

**Purpose**: Gets the versioning configuration of a bucket.

**Parameters**:

- `name` (string): Bucket name

**Returns**: Dictionary with versioning information (`enabled`, `mfa_delete`)

#### Object Operations - Core

##### `put_object(bucket, key, content, **options) -> None`

**Purpose**: Uploads an object to the specified bucket.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key (path)
- `content` (string|bytes): Object content
- `**options`: Additional options:
  - `content_type` (string): MIME type of the content
  - `metadata` (dict): Custom metadata key-value pairs
  - `tags` (dict): Object tags
  - `content_encoding` (string): Content encoding
  - `cache_control` (string): Cache control header
  - `expires` (string): Expiration date

##### `put_object_from_file(bucket, key, file_path, **options) -> None`

**Purpose**: Uploads a file to the specified bucket.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key (path)
- `file_path` (string): Local file path to upload
- `**options`: Same options as `put_object()`

##### `get_object(bucket, key) -> string`

**Purpose**: Downloads an object's content as a string.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Object content as string

**Raises**: Error if object doesn't exist

##### `get_object_to_file(bucket, key, file_path) -> None`

**Purpose**: Downloads an object directly to a local file.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `file_path` (string): Local file path to save to

##### `delete_object(bucket, key) -> None`

**Purpose**: Deletes a single object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key to delete

##### `delete_objects(bucket, keys) -> dict`

**Purpose**: Deletes multiple objects in a single request.

**Parameters**:

- `bucket` (string): Bucket name
- `keys` (list): List of object keys to delete

**Returns**: Dictionary with `deleted` (list) and `errors` (list) fields

#### Object Operations - Advanced

##### `copy_object(src_bucket, src_key, dst_bucket, dst_key, **options) -> None`

**Purpose**: Copies an object from one location to another.

**Parameters**:

- `src_bucket` (string): Source bucket name
- `src_key` (string): Source object key
- `dst_bucket` (string): Destination bucket name
- `dst_key` (string): Destination object key
- `**options`: Copy options (metadata, tags, etc.)

##### `move_object(src_bucket, src_key, dst_bucket, dst_key, **options) -> None`

**Purpose**: Moves an object (copy + delete source).

**Parameters**: Same as `copy_object()`

##### `list_objects(bucket, prefix="", delimiter="", max_keys=1000) -> dict`

**Purpose**: Lists objects in a bucket with optional filtering.

**Parameters**:

- `bucket` (string): Bucket name
- `prefix` (string, optional): Object key prefix filter
- `delimiter` (string, optional): Delimiter for grouping (e.g., "/" for folders)
- `max_keys` (int, optional): Maximum number of keys to return. Default: `1000`

**Returns**: Dictionary with object listing information:

- `contents` (list): List of object dictionaries
- `common_prefixes` (list): List of common prefixes (when delimiter is used)
- `is_truncated` (bool): Whether more results are available
- `next_marker` (string): Marker for next page of results

##### `get_object_info(bucket, key) -> dict`

**Purpose**: Gets metadata and properties of an object without downloading content.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary with object information (`size`, `last_modified`, `etag`, `content_type`, `metadata`)

##### `object_exists(bucket, key) -> bool`

**Purpose**: Checks if an object exists.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: `True` if object exists, `False` otherwise

#### Metadata and Tagging

##### `get_object_metadata(bucket, key) -> dict`

**Purpose**: Gets custom metadata for an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary of metadata key-value pairs

##### `set_object_metadata(bucket, key, metadata) -> None`

**Purpose**: Sets custom metadata for an object (requires copy operation).

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `metadata` (dict): Metadata key-value pairs

##### `get_object_tags(bucket, key) -> dict`

**Purpose**: Gets tags for an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns**: Dictionary of tag key-value pairs

##### `set_object_tags(bucket, key, tags) -> None`

**Purpose**: Sets tags for an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `tags` (dict): Tag key-value pairs

##### `delete_object_tags(bucket, key) -> None`

**Purpose**: Removes all tags from an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key

#### Pre-signed URLs

##### `presign_url(bucket, key, expires_in=3600, method="GET") -> string`

**Purpose**: Generates a pre-signed URL for temporary access to an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration in seconds. Default: `3600` (1 hour)
- `method` (string, optional): HTTP method ("GET" or "HEAD"). Default: `"GET"`

**Returns**: Pre-signed URL string

##### `presign_put_url(bucket, key, expires_in=3600, **options) -> string`

**Purpose**: Generates a pre-signed URL for uploading an object.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration in seconds. Default: `3600`
- `**options`: Upload constraints (content_type, metadata, etc.)

**Returns**: Pre-signed PUT URL string

##### `presign_post(bucket, key, expires_in=3600, **options) -> dict`

**Purpose**: Generates pre-signed POST data for browser uploads.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration in seconds. Default: `3600`
- `**options`: Upload constraints and conditions

**Returns**: Dictionary with `url` and `fields` for HTML form

#### Multi-part Upload

##### `create_multipart_upload(bucket, key, **options) -> string`

**Purpose**: Initiates a multipart upload for large files.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `**options`: Upload options (content_type, metadata, etc.)

**Returns**: Upload ID string

##### `upload_part(bucket, key, upload_id, part_number, content) -> dict`

**Purpose**: Uploads a single part of a multipart upload.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Upload ID from `create_multipart_upload()`
- `part_number` (int): Part number (1-10000)
- `content` (string|bytes): Part content

**Returns**: Dictionary with part information (`part_number`, `etag`)

##### `complete_multipart_upload(bucket, key, upload_id, parts) -> dict`

**Purpose**: Completes a multipart upload by combining all parts.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Upload ID
- `parts` (list): List of part dictionaries from `upload_part()`

**Returns**: Dictionary with upload result (`etag`, `location`)

##### `abort_multipart_upload(bucket, key, upload_id) -> None`

**Purpose**: Aborts a multipart upload and frees storage.

**Parameters**:

- `bucket` (string): Bucket name
- `key` (string): Object key
- `upload_id` (string): Upload ID to abort

##### `list_multipart_uploads(bucket, prefix="") -> list`

**Purpose**: Lists ongoing multipart uploads in a bucket.

**Parameters**:

- `bucket` (string): Bucket name
- `prefix` (string, optional): Object key prefix filter

**Returns**: List of upload dictionaries with upload information

### Basic Usage Examples

#### Basic File Operations

```python
load("s3", "create_client")

def main():
    s3 = create_client(region="us-east-1")
    
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

#### Service-Specific Configuration Examples

```python
# Cloudflare R2 example
r2_client = create_client(
    service_type="cloudflare_r2",
    endpoint="https://<account-id>.r2.cloudflarestorage.com",
    access_key="YOUR_R2_ACCESS_KEY",
    secret_key="YOUR_R2_SECRET_KEY",
    region="auto"
)

# MinIO example  
minio_client = create_client(
    service_type="minio",
    endpoint="http://localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    force_path_style=True,
    use_ssl=False
)

# Both clients use the same API
for client in [r2_client, minio_client]:
    client.put_object("test-bucket", "test.txt", "Hello World!")
    content = client.get_object("test-bucket", "test.txt")
    print("Content:", content)
```

#### Metadata and Tagging Example

```python
load("s3", "create_client")

def main():
    s3 = create_client()
    bucket = "metadata-demo"
    
    # Upload with metadata and tags
    s3.put_object(
        bucket,
        "document.pdf",
        "Document content...",
        content_type="application/pdf",
        metadata={
            "author": "Jane Doe",
            "department": "Engineering",
            "version": "2.1"
        },
        tags={
            "project": "alpha",
            "confidential": "true"
        }
    )
    
    # Retrieve metadata
    metadata = s3.get_object_metadata(bucket, "document.pdf")
    print("Author:", metadata.get("author"))
    
    # Get tags
    tags = s3.get_object_tags(bucket, "document.pdf")
    for key, value in tags.items():
        print("{}: {}".format(key, value))

main()
```

#### Error Handling Example

```python
load("s3", "create_client", "validate_bucket_name")

def safe_upload(s3, bucket_name, object_key, content):
    """Safely upload with validation and error handling"""
    
    # Validate bucket name
    if not validate_bucket_name(bucket_name):
        fail("Invalid bucket name: {}".format(bucket_name))
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        s3.create_bucket(bucket_name)
        print("Created bucket: {}".format(bucket_name))
    
    # Upload with error handling
    try:
        s3.put_object(bucket_name, object_key, content)
        print("Upload successful: s3://{}/{}".format(bucket_name, object_key))
    except Exception as e:
        fail("Upload failed: {}".format(e))

def main():
    s3 = create_client()
    safe_upload(s3, "my-safe-bucket", "test.txt", "Safe content")

main()
```

## Complete Usage Examples

The following examples have been moved to separate files for better organization:

- [`examples/basic_file_management.star`](examples/basic_file_management.star) - Basic file upload/download operations
- [`examples/website_deployment.star`](examples/website_deployment.star) - Static website deployment with CDN optimization
- [`examples/backup_system.star`](examples/backup_system.star) - Automated backup system with versioning
- [`examples/data_processing_pipeline.star`](examples/data_processing_pipeline.star) - ETL pipeline with S3 integration
- [`examples/multi_service_demo.star`](examples/multi_service_demo.star) - Working with multiple S3-compatible services
- [`examples/error_handling_best_practices.star`](examples/error_handling_best_practices.star) - Robust error handling patterns

Each example file contains complete, runnable Starlark code with detailed comments explaining the implementation.

## Implementation Architecture

### File Structure

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
├── config.go       # Configuration system with base package integration
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
    AccessKeyID     *base.ConfigOption[string]       // Access key ID
    SecretAccessKey *base.ConfigOption[base.Secret]  // Secret key (secure)
    SessionToken    *base.ConfigOption[string]       // Temporary session token
    
    // Service configuration
    Region          *base.ConfigOption[string]       // Region
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

# Authentication (compatible with AWS CLI/SDK)
export AWS_ACCESS_KEY_ID="YOUR_ACCESS_KEY"         # Access key ID
export AWS_SECRET_ACCESS_KEY="YOUR_SECRET_KEY"     # Secret access key
export AWS_SESSION_TOKEN="YOUR_SESSION_TOKEN"      # Session token (optional)
export AWS_DEFAULT_REGION="us-east-1"              # Region

# S3-specific configuration
export S3_FORCE_PATH_STYLE="false"                 # Path-style addressing
export S3_USE_SSL="true"                           # Enable SSL/TLS
export S3_PART_SIZE="5242880"                      # Multipart upload part size (5MB)
export S3_CONCURRENCY="3"                          # Concurrent operations

# Debug and monitoring
export S3_ENABLE_LOGGING="false"                   # Enable request logging
export S3_USER_AGENT="starlark-s3/1.0"            # Custom user agent
```

## Security & Performance

### Security Considerations

#### 1. Credential Management

- **Never log credentials**: Credentials are never exposed in error messages or logs
- **Secure storage**: Use `base.Secret` type for sensitive configuration values
- **Environment variables**: Support standard AWS credential chain
- **Automatic rotation**: Compatible with AWS credential rotation mechanisms

#### 2. Input Validation

- **Bucket names**: Validate according to AWS S3 naming rules
- **Object keys**: Sanitize to prevent path traversal and injection attacks
- **Size limits**: Enforce reasonable limits for uploads and downloads
- **Content validation**: Validate content types and encoding

#### 3. Network Security

- **HTTPS by default**: All communications use HTTPS unless explicitly disabled
- **Certificate validation**: Full SSL/TLS certificate chain validation
- **Request signing**: All requests signed with AWS Signature Version 4
- **Timeout protection**: Configurable timeouts prevent hanging connections

### Performance Optimizations

#### 1. Connection Management

- **Connection pooling**: HTTP connections are pooled and reused
- **Keep-alive**: Persistent connections for multiple requests
- **DNS caching**: Automatic DNS result caching for performance
- **Circuit breaker**: Automatic failure detection and recovery

#### 2. Upload/Download Optimization

- **Streaming**: Large files are streamed to minimize memory usage
- **Multipart uploads**: Automatic multipart uploads for large files
- **Concurrent operations**: Parallel uploads/downloads when beneficial
- **Resume capability**: Support for resuming interrupted transfers

#### 3. Caching Strategy

- **Metadata caching**: Client-side caching of frequently accessed metadata
- **Response caching**: Intelligent caching of list operations
- **Conditional requests**: Use ETags for conditional operations
- **Exponential backoff**: Intelligent retry with increasing delays

### Performance Targets

- **Throughput**: 1000+ small operations per second
- **Latency**: <100ms for metadata operations
- **Memory efficiency**: <50MB memory usage for typical workloads
- **Large file support**: Stream files of any size without memory issues
- **Concurrent operations**: Support for 100+ concurrent requests

## Migration Guide

### From AWS CLI

```bash
# AWS CLI command
aws s3 cp file.txt s3://bucket/path/

# Starlark equivalent
load("s3", "create_client")
load("file", "read")

s3 = create_client()
content = read("file.txt")
s3.put_object("bucket", "path/file.txt", content)
```

### From boto3 (Python)

```python
# boto3 Python code
import boto3
s3 = boto3.client('s3')
s3.create_bucket(Bucket='bucket-name')
s3.put_object(Bucket='bucket', Key='key', Body=data)

# Starlark equivalent
load("s3", "create_client")
s3 = create_client()
s3.create_bucket('bucket-name')
s3.put_object('bucket', 'key', data)
```

### From MinIO Client

```go
// MinIO Go client
minioClient, _ := minio.New("localhost:9000", &minio.Options{
    Creds: credentials.NewStaticV4("minioadmin", "minioadmin", ""),
})

// Starlark equivalent
load("s3", "create_client")
s3 = create_client(
    service_type="minio",
    endpoint="http://localhost:9000", 
    access_key="minioadmin",
    secret_key="minioadmin",
    force_path_style=True,
    use_ssl=False
)
```
