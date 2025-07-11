# 🗂️ S3 Module for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/s3.svg)](https://pkg.go.dev/github.com/starpkg/s3)
[![Go Report Card](https://goreportcard.com/badge/github.com/starpkg/s3)](https://goreportcard.com/report/github.com/starpkg/s3)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Universal S3-compatible storage operations for Starlark scripts - seamlessly connect to any S3 service!**

The S3 module provides a comprehensive, easy-to-use interface for interacting with S3-compatible storage services from Starlark scripts. It supports Amazon S3, MinIO, DigitalOcean Spaces, Cloudflare R2, and many other S3-compatible services.

## ✨ Features

- **🌐 Universal Compatibility**: Works with AWS S3, MinIO, DigitalOcean Spaces, Cloudflare R2, and other S3-compatible services
- **🔒 Secure Authentication**: Supports multiple authentication methods including IAM, access keys, and session tokens
- **🪣 Bucket Operations**: Create, delete, list, and manage buckets with proper validation
- **📁 Object Operations**: Upload, download, delete, and list objects with metadata support
- **🛠️ Utility Functions**: URL parsing, bucket name validation, and service configuration helpers
- **⚡ High Performance**: Built on AWS SDK v2 with configurable concurrency and retry policies
- **🔍 Smart Configuration**: Auto-detection of service types and intelligent default settings
- **🎯 Starlark Native**: Designed specifically for Starlark with proper error handling and type safety

## 🚀 Quick Start

### Basic Usage

```python
# Load the S3 module
load("s3", "create_client")

# Create a client for AWS S3
client = create_client(
    service_type="aws",
    region="us-west-2",
    access_key="your-access-key",
    secret_key="your-secret-key",
)

# Create a bucket
client.create_bucket("my-bucket")

# Upload a file
client.put_object("my-bucket", "hello.txt", "Hello, World!")

# Download a file
content = client.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"

# List objects
objects = client.list_objects("my-bucket")
for obj in objects["contents"]:
    print(obj["key"], obj["size"])
```

### MinIO Example

```python
load("s3", "create_client")

# Create a client for MinIO
client = create_client(
    service_type="minio",
    endpoint="localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    use_ssl=False,
)

# Check if bucket exists
if not client.bucket_exists("test-bucket"):
    client.create_bucket("test-bucket")

# Upload with metadata
client.put_object(
    "test-bucket", 
    "data.json", 
    '{"message": "Hello from Starlark!"}',
    content_type="application/json",
    metadata={"source": "starlark-script"}
)
```

## 🔧 Configuration

### Client Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `service_type` | string | `"auto"` | S3 service type (aws, minio, digitalocean, etc.) |
| `access_key` | string | `""` | S3 access key ID |
| `secret_key` | string | `""` | S3 secret access key |
| `session_token` | string | `""` | S3 session token (for temporary credentials) |
| `region` | string | `"us-east-1"` | S3 region |
| `endpoint` | string | `""` | Custom S3 endpoint URL |
| `force_path_style` | bool | `false` | Force path-style addressing |
| `use_ssl` | bool | `true` | Use SSL/TLS for connections |
| `timeout` | int | `30` | Request timeout in seconds |
| `max_retries` | int | `3` | Maximum number of retry attempts |
| `part_size` | int | `5242880` | Multipart upload part size (5MB) |
| `concurrency` | int | `3` | Number of concurrent uploads |
| `enable_logging` | bool | `false` | Enable debug logging |
| `user_agent` | string | `"starlark-s3/1.0"` | Custom user agent string |

### Supported Services

The module supports these S3-compatible services:

- **AWS S3** (`service_type="aws"`)
- **MinIO** (`service_type="minio"`)
- **DigitalOcean Spaces** (`service_type="digitalocean"`)
- **Linode Object Storage** (`service_type="linode"`)
- **Wasabi** (`service_type="wasabi"`)
- **Backblaze B2** (`service_type="backblaze"`)
- **Cloudflare R2** (`service_type="cloudflare"`)
- **Scaleway** (`service_type="scaleway"`)
- **Alibaba Cloud OSS** (`service_type="alibaba"`)
- **Google Cloud Storage** (`service_type="google"`)
- **Oracle Cloud** (`service_type="oracle"`)
- **IBM Cloud** (`service_type="ibm"`)
- **Custom** (`service_type="custom"`)

## 📚 API Reference

### Client Creation

#### `create_client(**kwargs)`
Creates a new S3 client with the specified configuration.

**Parameters:**
- All configuration options listed above

**Returns:**
- S3 client object with bucket and object operation methods

### Bucket Operations

#### `client.create_bucket(bucket, region=None)`
Creates a new S3 bucket.

**Parameters:**
- `bucket` (string): Bucket name
- `region` (string, optional): Bucket region

#### `client.delete_bucket(bucket, force=False)`
Deletes an S3 bucket.

**Parameters:**
- `bucket` (string): Bucket name
- `force` (bool): If True, deletes all objects first

#### `client.list_buckets()`
Lists all buckets in the account.

**Returns:**
- List of bucket information dictionaries

#### `client.bucket_exists(bucket)`
Checks if a bucket exists.

**Parameters:**
- `bucket` (string): Bucket name

**Returns:**
- Boolean indicating if bucket exists

#### `client.get_bucket_location(bucket)`
Gets the location/region of a bucket.

**Parameters:**
- `bucket` (string): Bucket name

**Returns:**
- String containing bucket region

### Object Operations

#### `client.put_object(bucket, key, body, **kwargs)`
Uploads an object to S3.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key
- `body` (string): Object content
- `content_type` (string, optional): MIME type
- `metadata` (dict, optional): Object metadata
- `cache_control` (string, optional): Cache control header
- `content_encoding` (string, optional): Content encoding

#### `client.get_object(bucket, key)`
Downloads an object from S3.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns:**
- String containing object content

#### `client.delete_object(bucket, key)`
Deletes an object from S3.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key

#### `client.list_objects(bucket, **kwargs)`
Lists objects in a bucket.

**Parameters:**
- `bucket` (string): Bucket name
- `prefix` (string, optional): Object key prefix
- `delimiter` (string, optional): Delimiter for grouping
- `max_keys` (int, optional): Maximum number of keys to return
- `continuation_token` (string, optional): Pagination token

**Returns:**
- Dictionary containing object list and metadata

#### `client.object_exists(bucket, key)`
Checks if an object exists.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns:**
- Boolean indicating if object exists

#### `client.get_object_info(bucket, key)`
Gets metadata about an object.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns:**
- Dictionary containing object metadata

### Utility Functions

#### `parse_s3_url(url)`
Parses an S3 URL into bucket and key components.

**Parameters:**
- `url` (string): S3 URL (s3://, http://, or https://)

**Returns:**
- Dictionary with `bucket` and `key` fields

#### `generate_s3_url(bucket, key)`
Generates a standard S3 URL.

**Parameters:**
- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns:**
- String containing S3 URL

#### `validate_bucket_name(name)`
Validates an S3 bucket name.

**Parameters:**
- `name` (string): Bucket name to validate

**Returns:**
- Boolean indicating if name is valid

#### `validate_object_key(key)`
Validates an S3 object key.

**Parameters:**
- `key` (string): Object key to validate

**Returns:**
- Boolean indicating if key is valid

#### `get_supported_services()`
Returns a list of supported S3 services.

**Returns:**
- List of supported service type strings

## 🎯 Examples

### Working with Different Services

```python
load("s3", "create_client")

# AWS S3
aws_client = create_client(
    service_type="aws",
    region="us-west-2",
    access_key="AKIAIOSFODNN7EXAMPLE",
    secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
)

# MinIO
minio_client = create_client(
    service_type="minio",
    endpoint="localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin",
    use_ssl=False,
)

# DigitalOcean Spaces
do_client = create_client(
    service_type="digitalocean",
    region="nyc3",
    access_key="your-spaces-key",
    secret_key="your-spaces-secret",
)

# Cloudflare R2
r2_client = create_client(
    service_type="cloudflare",
    endpoint="your-account-id.r2.cloudflarestorage.com",
    access_key="your-r2-access-key",
    secret_key="your-r2-secret-key",
)
```

### Advanced Usage

```python
load("s3", "create_client", "parse_s3_url", "validate_bucket_name")

# Create client with custom configuration
client = create_client(
    service_type="aws",
    region="eu-west-1",
    access_key="your-access-key",
    secret_key="your-secret-key",
    timeout=60,
    max_retries=5,
    part_size=10485760,  # 10MB parts
    concurrency=10,      # 10 concurrent uploads
    enable_logging=True,
)

# Validate bucket name before creating
bucket_name = "my-new-bucket"
if validate_bucket_name(bucket_name):
    if not client.bucket_exists(bucket_name):
        client.create_bucket(bucket_name, region="eu-west-1")
        print(f"Created bucket: {bucket_name}")
    else:
        print(f"Bucket already exists: {bucket_name}")
else:
    print(f"Invalid bucket name: {bucket_name}")

# Upload file with metadata
client.put_object(
    bucket_name,
    "documents/important.pdf",
    file_content,
    content_type="application/pdf",
    metadata={
        "author": "John Doe",
        "department": "Engineering",
        "classification": "internal",
    },
    cache_control="max-age=3600",
)

# Parse S3 URLs
s3_url = "s3://my-bucket/path/to/file.txt"
parsed = parse_s3_url(s3_url)
print(f"Bucket: {parsed['bucket']}, Key: {parsed['key']}")

# List objects with pagination
continuation_token = None
while True:
    result = client.list_objects(
        bucket_name,
        prefix="documents/",
        max_keys=100,
        continuation_token=continuation_token,
    )
    
    for obj in result["contents"]:
        print(f"Object: {obj['key']} ({obj['size']} bytes)")
    
    if not result["is_truncated"]:
        break
    
    continuation_token = result.get("next_marker")
```

### Error Handling

```python
load("s3", "create_client")

def safe_s3_operation():
    try:
        client = create_client(
            service_type="aws",
            region="us-west-2",
            access_key="your-access-key",
            secret_key="your-secret-key",
        )
        
        # Try to create bucket
        client.create_bucket("my-test-bucket")
        print("Bucket created successfully")
        
        # Try to upload object
        client.put_object("my-test-bucket", "test.txt", "Hello World")
        print("Object uploaded successfully")
        
    except Exception as e:
        print(f"S3 operation failed: {e}")
        return False
    
    return True

# Run the safe operation
if safe_s3_operation():
    print("All operations completed successfully")
else:
    print("Some operations failed")
```

## 🧪 Testing

Run the test suite:

```bash
go test -v
```

The test suite includes:
- Client creation and configuration tests
- Utility function tests
- API method availability tests
- URL parsing tests
- Bucket and object operation interface tests

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📞 Support

For support, please open an issue on the GitHub repository.

---

**Made with ❤️ for the Starlark community** 