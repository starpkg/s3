# 🗂️ S3 Module for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/s3.svg)](https://pkg.go.dev/github.com/starpkg/s3)
[![Go Report Card](https://goreportcard.com/badge/github.com/starpkg/s3)](https://goreportcard.com/report/github.com/starpkg/s3)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Universal S3-compatible storage operations for Starlark scripts - seamlessly connect to any S3 service!**

The S3 module provides a comprehensive, easy-to-use interface for interacting with S3-compatible storage services from Starlark scripts. It supports Amazon S3, MinIO, DigitalOcean Spaces, Cloudflare R2, and many other S3-compatible services with advanced features like file operations, metadata management, and multi-provider URL handling.

## ✨ Features

- **🌐 Universal Compatibility**: Works with AWS S3, MinIO, DigitalOcean Spaces, Cloudflare R2, and other S3-compatible services
- **🔒 Secure Configuration**: Module-level configuration with secret handling and environment variable support
- **🪣 Advanced Bucket Operations**: Create, delete, list, and get comprehensive bucket information
- **📁 Enhanced Object Operations**: Upload, download, delete, list, copy, and manage object metadata and properties
- **📂 Direct File Operations**: Upload and download files directly from filesystem for better performance
- **🏷️ Metadata & Tags**: Full support for object metadata, tags, content-type, cache-control, and more
- **🔗 Smart URL Handling**: Multi-provider URL parsing and generation with automatic service detection
- **🛠️ Rich Utility Functions**: Bucket validation, object key validation, and service configuration helpers
- **⚡ High Performance**: Built on AWS SDK v2 with configurable concurrency and retry policies
- **🧠 Smart Provider Detection**: Intelligent, pluggable auto-detection of service providers based on endpoints, regions, and access key patterns
- **🔍 Intelligent Configuration**: Environment variable integration and smart defaults
- **🎯 Starlark Native**: Designed specifically for Starlark with proper error handling and type safety



## 🚀 Quick Start

### Basic Usage

```python
# Load the S3 module
load("s3", "create_client")

# Create a client with smart provider detection
client = create_client(
    region="us-west-2",
    access_key="AKIAIOSFODNN7EXAMPLE",  # Automatically detects AWS S3
    secret_key="your-secret-key",
)

# Create a bucket
client.create_bucket("my-bucket")

# Upload a file
client.put_object("my-bucket", "hello.txt", "Hello, World!")

# Download a file
content = client.get_object("my-bucket", "hello.txt")
print(content)  # "Hello, World!"

# List objects (returns a list directly)
objects = client.list_objects("my-bucket")
for obj in objects:
    print(obj["key"], obj["size"])

# Generate a temporary download link
download_url = client.presign_url("my-bucket", "hello.txt", expires_in=3600)
print(f"Temporary download URL: {download_url}")
```

### File Operations

```python
# Upload a file directly from filesystem
client.put_object_file(
    "my-bucket", 
    "data/backup.zip", 
    "/local/path/backup.zip",
    content_type="application/zip"
)

# Download a file directly to filesystem
client.get_object_file("my-bucket", "data/backup.zip", "/local/path/downloaded.zip")
```

### Enhanced Object Management

```python
# Set comprehensive object properties
client.set_object_info(
    "my-bucket",
    "document.pdf",
    content_type="application/pdf",
    cache_control="max-age=3600",
    content_encoding="gzip",
    content_disposition="attachment; filename=document.pdf",
    content_language="en-US",
    expires="2024-12-31T23:59:59Z",
    metadata={"author": "John Doe", "version": "1.0"},
    tags={"project": "alpha", "department": "engineering"}
)

# Copy objects with metadata changes
client.copy_object(
    "source-bucket", "source/file.txt",
    "dest-bucket", "destination/file.txt",
    content_type="text/plain",
    metadata={"copied": "true"}
)
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

# Get comprehensive bucket information
bucket_info = client.get_bucket_info("test-bucket")
print(f"Bucket: {bucket_info['name']}")
print(f"Region: {bucket_info['region']}")
print(f"Created: {bucket_info['creation_date']}")
print(f"Versioning: {bucket_info['versioning_status']}")
print(f"Object count: {bucket_info['object_count']}")
print(f"Total size: {bucket_info['total_size']} bytes")
```

### Working with Time Objects

The S3 module returns time information as proper time.Time objects (from the `time` module) instead of strings. This provides better type safety and more convenient time operations:

```python
load("s3", "create_client")
load("time", "now")

client = create_client(
    access_key="your-access-key",
    secret_key="your-secret-key",
    region="us-west-2"
)

# Get bucket information with time objects
bucket_info = client.get_bucket_info("my-bucket")
creation_time = bucket_info["creation_date"]

# Time objects can be compared directly
current_time = now()
if creation_time < current_time:
    print("Bucket was created in the past")

# Get object information with time objects
obj_info = client.get_object_info("my-bucket", "my-file.txt")
last_modified = obj_info["last_modified"]

# Format time objects for display
print(f"Object last modified: {last_modified.format('2006-01-02 15:04:05')}")

# Calculate time differences
age_seconds = (current_time - last_modified).seconds
print(f"File is {age_seconds} seconds old")
```

## 🧠 Smart Provider Detection

The S3 module features an intelligent, pluggable provider detection system that automatically identifies the correct S3-compatible service based on configuration hints. This eliminates the need to explicitly specify `service_type` in most cases.

### Automatic Detection

Simply omit the `service_type` parameter or set it to `"auto"` to enable smart detection:

```python
load("s3", "create_client")

# Smart detection based on endpoint
client = create_client(
    endpoint="https://e0ed38ec5a87ac84d936841eee7336b2.r2.cloudflarestorage.com",
    access_key="your-key",
    secret_key="your-secret"
)
# ✅ Automatically detects Cloudflare R2

# Smart detection based on region
client = create_client(
    region="auto",  # R2-specific region
    access_key="f1889d933799dc332549e6671a042e36",  # 32-char hex pattern
    secret_key="your-secret"
)
# ✅ Automatically detects Cloudflare R2

# Smart detection based on access key pattern
client = create_client(
    access_key="AKIAIOSFODNN7EXAMPLE",  # AWS pattern
    secret_key="your-secret",
    region="us-west-2"
)
# ✅ Automatically detects AWS S3

# Smart detection for MinIO
client = create_client(
    access_key="minioadmin",  # Default MinIO credentials
    secret_key="minioadmin",
    endpoint="localhost:9000"
)
# ✅ Automatically detects MinIO
```

### Detection Rules & Priority

The detection system uses a priority-based rule engine that evaluates multiple factors:

1. **Endpoint Patterns** (Highest Priority - 5-10)
   - `amazonaws.com` → AWS S3
   - `r2.cloudflarestorage.com` → Cloudflare R2
   - `digitaloceanspaces.com` → DigitalOcean Spaces
   - `linodeobjects.com` → Linode Object Storage
   - `wasabisys.com` → Wasabi
   - `backblazeb2.com` → Backblaze B2
   - `scw.cloud` → Scaleway
   - `aliyuncs.com` → Alibaba Cloud OSS
   - `googleapis.com` → Google Cloud Storage
   - `oraclecloud.com` → Oracle Cloud Infrastructure
   - `cloud-object-storage.appdomain.cloud` → IBM Cloud

2. **Special Region Indicators** (Priority 15)
   - `region="auto"` → Cloudflare R2

3. **Access Key Patterns** (Priority 20-25)
   - `AKIA[A-Z0-9]{16}` → AWS S3 (standard keys)
   - `ASIA[A-Z0-9]{16}` → AWS S3 (temporary keys)
   - `[0-9a-fA-F]{32}` → Cloudflare R2 (32-char hex)

4. **Region Formats** (Priority 30-35)
   - `[a-z]{2,3}-[a-z]+-\d+` → AWS S3 (e.g., us-west-2)
   - `nyc1`, `nyc3`, `fra1`, etc. → DigitalOcean Spaces

5. **Default Credentials** (Priority 40)
   - `minioadmin`, `minio` → MinIO

6. **Endpoint Characteristics** (Priority 50+)
   - Localhost with port → MinIO
   - Domain containing "min.io" → MinIO

### Supported Providers

All major S3-compatible providers are supported with automatic detection:

| Provider | Detection Criteria | Example |
|----------|-------------------|---------|
| **AWS S3** | `amazonaws.com` endpoints, `AKIA`/`ASIA` keys, AWS regions | `us-west-2`, `AKIAIOSFODNN7EXAMPLE` |
| **Cloudflare R2** | `r2.cloudflarestorage.com`, `region="auto"`, 32-char hex keys | `f1889d933799dc332549e6671a042e36` |
| **MinIO** | Default credentials, localhost endpoints, `min.io` domains | `minioadmin`, `localhost:9000` |
| **DigitalOcean Spaces** | `digitaloceanspaces.com`, DO region codes | `nyc3`, `fra1`, `ams3` |
| **Linode Object Storage** | `linodeobjects.com` endpoints | `us-east-1.linodeobjects.com` |
| **Wasabi** | `wasabisys.com` endpoints | `s3.us-east-1.wasabisys.com` |
| **Backblaze B2** | `backblazeb2.com` endpoints | `s3.us-west-000.backblazeb2.com` |
| **Scaleway** | `scw.cloud` endpoints | `s3.fr-par.scw.cloud` |
| **Alibaba Cloud OSS** | `aliyuncs.com` endpoints | `oss-cn-hangzhou.aliyuncs.com` |
| **Google Cloud Storage** | `googleapis.com` endpoints | `storage.googleapis.com` |
| **Oracle Cloud** | `oraclecloud.com` endpoints | `namespace.compat.objectstorage.us-ashburn-1.oraclecloud.com` |
| **IBM Cloud** | `cloud-object-storage.appdomain.cloud` | `s3.us-south.cloud-object-storage.appdomain.cloud` |

### Override Detection

You can always override automatic detection by explicitly specifying `service_type`:

```python
# Force MinIO even with AWS-like credentials
client = create_client(
    service_type="minio",  # Explicit override
    access_key="AKIAIOSFODNN7EXAMPLE",  # Would normally detect AWS
    secret_key="your-secret",
    endpoint="localhost:9000"
)
```

### Legacy Script Migration

Existing scripts automatically benefit from smart detection without any changes:

```python
# Old script - still works perfectly
client = create_client(
    access_key="AKIAIOSFODNN7EXAMPLE",
    secret_key="your-secret",
    region="us-west-2"
)
# ✅ Automatically detects AWS S3 - no migration needed!
```

## 🔧 Configuration

### Module-Level Configuration

The S3 module supports module-level configuration with environment variable integration:

```python
# Environment variables are automatically detected:
# AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN, AWS_DEFAULT_REGION
# S3_SERVICE_TYPE, S3_ENDPOINT, S3_USE_SSL, etc.

# Create client using module defaults + overrides
client = create_client(
    # Only specify what you want to override
    service_type="aws",
    region="eu-west-1"
    # access_key and secret_key can come from environment
)
```

### Client Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `service_type` | string | `"auto"` | S3 service type (aws, minio, digitalocean, etc.) |
| `access_key` | string | `""` | S3 access key ID (secret) |
| `secret_key` | string | `""` | S3 secret access key (secret) |
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

The module supports these S3-compatible services with automatic endpoint detection:

| Service | Provider Constant | Service Type String | Default Region | Description |
|---------|------------------|-------------------|---------------|-------------|
| Amazon S3 | `ProviderAWS` | `"aws"` | `us-east-1` | AWS Simple Storage Service |
| MinIO | `ProviderMinIO` | `"minio"` | `us-east-1` | High Performance Object Storage |
| DigitalOcean Spaces | `ProviderDigitalOcean` | `"digitalocean"` | `nyc3` | DigitalOcean's object storage |
| Linode Object Storage | `ProviderLinode` | `"linode"` | `us-east-1` | Linode's S3-compatible storage |
| Wasabi Hot Storage | `ProviderWasabi` | `"wasabi"` | `us-east-1` | Low-cost cloud storage |
| Backblaze B2 | `ProviderBackblaze` | `"backblaze"` | `us-west-000` | Backblaze B2 Cloud Storage |
| Cloudflare R2 | `ProviderCloudflare` | `"cloudflare"` | `auto` | Cloudflare R2 Storage |
| Scaleway Object Storage | `ProviderScaleway` | `"scaleway"` | `fr-par` | Scaleway's object storage |
| Alibaba Cloud OSS | `ProviderAlibaba` | `"alibaba"` | `oss-cn-hangzhou` | Alibaba Cloud Object Storage |
| Google Cloud Storage | `ProviderGoogle` | `"google"` | `us-central1` | Google Cloud Storage |
| Oracle Cloud | `ProviderOracle` | `"oracle"` | `us-ashburn-1` | Oracle Cloud Infrastructure |
| IBM Cloud | `ProviderIBM` | `"ibm"` | `us-south` | IBM Cloud Object Storage |
| Custom Provider | `ProviderCustom` | `"custom"` | `us-east-1` | Generic S3-compatible service |

### 📝 Provider-Specific Notes

#### Alibaba Cloud OSS Considerations

When working with Alibaba Cloud OSS, please note the following based on the [official CopyObject documentation](https://help.aliyun.com/zh/oss/developer-reference/copyobject):

- **Copy Operations**: Complex metadata modifications during copy operations may require specific header signing that differs from standard AWS S3. Our tests use simplified copy operations for maximum compatibility.
- **Permissions**: CopyObject requires both `oss:GetObject` (source) and `oss:PutObject` (destination) permissions
- **Size Limitations**: 
  - Same bucket copies: Objects can be larger than 5GB
  - Cross-bucket copies: Objects must be ≤5GB  
  - Storage type changes: Objects must be ≤1GB
- **Chinese Character Support**: Full UTF-8 support for object keys and metadata values
- **Regional Endpoints**: Use region-specific endpoints like `oss-cn-hangzhou.aliyuncs.com`

#### Cloudflare R2 Considerations

- **No Object Tagging**: R2 doesn't support S3 object tagging operations
- **Region Setting**: Use `region="auto"` for R2 endpoints
- **Account ID**: Include your Cloudflare account ID in the endpoint URL

#### AWS S3 Considerations  

- **Complete Compatibility**: Full feature support including advanced metadata, tagging, and versioning
- **Regional Optimization**: Choose regions close to your application for better performance
- **Cost Optimization**: Consider storage classes and lifecycle policies for cost management

## ⚠️ Provider Feature Compatibility

While the S3 module provides a universal interface, some advanced features may not be supported by all providers:

| Feature | AWS S3 | Alibaba OSS | Cloudflare R2 | DigitalOcean | MinIO | Notes |
|---------|---------|-------------|---------------|--------------|-------|-------|
| **Object Tagging** | ✅ | ✅ | ❌ | ✅ | ✅ | R2 doesn't support S3 object tagging |
| **Bucket Versioning** | ✅ | ✅ | ❌ | ✅ | ✅ | R2 has limited versioning support |
| **Bucket Policies** | ✅ | ✅ | ✅ | ✅ | ✅ | All providers support basic policies |
| **Object Lock** | ✅ | ✅ | ❌ | ❌ | ✅ | Enterprise compliance feature |
| **Lifecycle Management** | ✅ | ✅ | ❌ | ✅ | ✅ | Automatic object expiration |
| **Multipart Upload** | ✅ | ✅ | ✅ | ✅ | ✅ | Large file upload optimization |
| **Presigned URLs** | ✅ | ✅ | ✅ | ✅ | ✅ | Temporary access URLs |
| **Server-Side Encryption** | ✅ | ✅ | ✅ | ✅ | ✅ | Data encryption at rest |
| **Cross-Region Replication** | ✅ | ✅ | ❌ | ❌ | ❌ | Geographic data distribution |
| **Event Notifications** | ✅ | ✅ | ❌ | ❌ | ✅ | Webhook/queue integration |
| **Access Logging** | ✅ | ✅ | ❌ | ✅ | ❌ | Request access logs |
| **Transfer Acceleration** | ✅ | ❌ | ✅ | ❌ | ❌ | Global edge acceleration |

### 🔧 Feature Usage Guidelines

When using advanced features, consider provider compatibility:

```python
# Safe approach - Check provider before using advanced features
if client.get_provider_type() == "aws":
    # Use advanced AWS features
    client.put_object(bucket, key, content, tags={"env": "prod"})
elif client.get_provider_type() == "cloudflare":
    # Use basic features for R2
    client.put_object(bucket, key, content)  # No tags
```

### 📋 Graceful Degradation

The module handles unsupported features gracefully:
- **Object Tagging**: Silently ignored on incompatible providers
- **Advanced Metadata**: Basic metadata preserved, advanced headers may be dropped
- **Error Handling**: Clear error messages for unsupported operations

### Provider Integration Process

To add support for a new S3-compatible service provider, follow these steps:

#### 1. Add Provider Constants

Define a new provider constant in `provider.go`:

```go
const (
    // ... existing providers ...
    ProviderNewService = "newservice"
)
```

#### 2. Configure Provider Settings

Add a new provider configuration in the `providerConfigs` map:

```go
ProviderNewService: {
    Name:                  ProviderNewService,
    DisplayName:           "New S3 Service",
    DefaultRegion:         "us-east-1",
    DefaultPort:           "443",
    ForcePathStyle:        false,
    URLStyle:              URLStyleVirtualHosted, // or URLStylePath, URLStyleBoth
    EndpointPattern:       "s3.{region}.newservice.com",
    SupportsVirtualHosted: true,
    SupportsPathStyle:     false,
    
    // URL patterns for parsing service URLs
    URLPatterns: []URLPattern{
        {
            Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.newservice\.com/`),
            ParseFunc: parseVirtualHostedURL,
        },
        {
            Pattern:   regexp.MustCompile(`^https?://s3\.[^/]+\.newservice\.com/`),
            ParseFunc: parsePathStyleURL,
        },
    },
    
    // URL generation function
    GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
        return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.newservice.com", false)
    },
},
```

#### 3. Test Integration

Create test cases to verify URL parsing and generation:

```go
// Test URL parsing
testCases := []struct {
    url      string
    provider string
    bucket   string
    key      string
}{
    {
        url:      "https://bucket.s3.region.newservice.com/file.txt",
        provider: "newservice",
        bucket:   "bucket", 
        key:      "file.txt",
    },
}
```

#### 4. Update Documentation

Add the new provider to the supported services table above and create usage examples.

#### Integration Requirements

For successful integration, a new provider must implement:

- **URL Pattern Recognition**: Regex patterns to identify and parse provider-specific URLs
- **Endpoint Generation**: Logic to generate correct endpoint URLs based on region and configuration
- **Authentication Support**: Compatible with AWS SDK v2 authentication mechanisms
- **Standard S3 API**: Support for core S3 operations (bucket and object management)

#### Provider-Specific Features

Some providers may have unique features or limitations:

- **Cloudflare R2**: Requires account ID in endpoint pattern
- **Oracle Cloud**: Requires namespace in endpoint pattern  
- **Google Cloud Storage**: Uses path-style addressing only
- **MinIO**: Typically uses custom endpoints and path-style addressing

Example implementation can be found in the existing provider configurations in `provider.go`.

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

- List of bucket information dictionaries, each containing:
  - `name` (string): Bucket name
  - `creation_date` (time.Time): Creation timestamp
  - `region` (string): Bucket region
  - `location` (string): Bucket location
  - `versioning_status` (string): Versioning configuration
  - `public_access_blocked` (bool): Public access block status
  - `has_policy` (bool): Whether bucket has a policy
  - `has_cors` (bool): Whether bucket has CORS configuration
  - `encryption_enabled` (bool): Whether encryption is enabled
  - `encryption_type` (string): Type of encryption used
  - `object_count` (int): Number of objects in bucket
  - `total_size` (int): Total size of all objects in bytes
  - `storage_class` (string): Default storage class
  - `tags` (dict): Bucket tags
  - `owner` (string): Bucket owner
  - `bucket_type` (string): Type of bucket

#### `client.bucket_exists(bucket)`

Checks if a bucket exists.

**Parameters:**

- `bucket` (string): Bucket name

**Returns:**

- Boolean indicating if bucket exists

#### `client.get_bucket_info(bucket)`

Gets comprehensive information about a bucket.

**Parameters:**

- `bucket` (string): Bucket name

**Returns:**

- Dictionary containing comprehensive bucket information:
  - `name` (string): Bucket name
  - `creation_date` (time.Time): Creation timestamp
  - `region` (string): Bucket region
  - `location` (string): Bucket location
  - `versioning_status` (string): Versioning configuration
  - `public_access_blocked` (bool): Public access block status
  - `has_policy` (bool): Whether bucket has a policy
  - `has_cors` (bool): Whether bucket has CORS configuration
  - `encryption_enabled` (bool): Whether encryption is enabled
  - `encryption_type` (string): Type of encryption used
  - `object_count` (int): Number of objects in bucket
  - `total_size` (int): Total size of all objects in bytes
  - `storage_class` (string): Default storage class
  - `tags` (dict): Bucket tags
  - `owner` (string): Bucket owner
  - `bucket_type` (string): Type of bucket

### Object Operations

#### `client.put_object(bucket, key, content, **kwargs)`

Uploads an object to S3.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `content` (string): Object content
- `content_type` (string, optional): MIME type
- `metadata` (dict, optional): Object metadata
- `tags` (dict, optional): Object tags
- `cache_control` (string, optional): Cache control header
- `content_encoding` (string, optional): Content encoding
- `expires` (string, optional): Expiration date (RFC3339 format)

#### `client.put_object_file(bucket, key, file_path, **kwargs)`

Uploads a file directly from filesystem to S3.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `file_path` (string): Local file path to upload
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

#### `client.get_object_file(bucket, key, file_path)`

Downloads an object from S3 directly to filesystem.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `file_path` (string): Local file path to save to

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

**Returns:**

- List of object info dictionaries, each containing:
  - `key` (string): Object key
  - `size` (int): Object size in bytes
  - `last_modified` (time.Time): Last modification timestamp
  - `etag` (string): Entity tag (ETag) of the object
  - `content_type` (string): MIME type of the object
  - `content_encoding` (string): Content encoding
  - `content_disposition` (string): Content disposition
  - `content_language` (string): Content language
  - `cache_control` (string): Cache control header
  - `expires` (time.Time): Expiration date
  - `storage_class` (string): Storage class
  - `version_id` (string): Version ID
  - `is_latest` (bool): Whether this is the latest version
  - `owner` (string): Object owner
  - `metadata` (dict): User-defined metadata key-value pairs
  - `tags` (dict): Object tags

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

- Dictionary containing comprehensive object metadata:
  - `key` (string): Object key
  - `size` (int): Object size in bytes
  - `last_modified` (time.Time): Last modification timestamp
  - `etag` (string): Entity tag (ETag) of the object
  - `content_type` (string): MIME type of the object
  - `content_encoding` (string): Content encoding
  - `content_disposition` (string): Content disposition
  - `content_language` (string): Content language
  - `cache_control` (string): Cache control header
  - `expires` (time.Time): Expiration date
  - `storage_class` (string): Storage class
  - `version_id` (string): Version ID
  - `is_latest` (bool): Whether this is the latest version
  - `owner` (string): Object owner
  - `metadata` (dict): User-defined metadata key-value pairs
  - `tags` (dict): Object tags

#### `client.set_object_info(bucket, key, **kwargs)`

Sets comprehensive properties for an object by copying it with new metadata.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `metadata` (dict, optional): Object metadata
- `tags` (dict, optional): Object tags
- `content_type` (string, optional): MIME type
- `cache_control` (string, optional): Cache control header
- `content_encoding` (string, optional): Content encoding
- `content_disposition` (string, optional): Content disposition header
- `content_language` (string, optional): Content language
- `expires` (string, optional): Expiration date (RFC3339 format)

#### `client.copy_object(src_bucket, src_key, dst_bucket, dst_key, **kwargs)`

Copies an object from one location to another.

**Parameters:**

- `src_bucket` (string): Source bucket name
- `src_key` (string): Source object key
- `dst_bucket` (string): Destination bucket name
- `dst_key` (string): Destination object key
- `content_type` (string, optional): New MIME type
- `metadata` (dict, optional): New object metadata
- `cache_control` (string, optional): New cache control header
- `content_encoding` (string, optional): New content encoding

#### `client.presign_url(bucket, key, expires_in=3600, method="GET")`

Generates a pre-signed URL for temporary access to an object.

**Purpose**: Creates a temporary URL that allows access to private objects without requiring AWS credentials. Useful for sharing files securely or enabling direct browser uploads/downloads.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `expires_in` (int, optional): URL expiration time in seconds. Default: `3600` (1 hour)
- `method` (string, optional): HTTP method for the URL. Supported: `"GET"`, `"HEAD"`. Default: `"GET"`

**Returns:**

- String containing the pre-signed URL

**Examples:**

```python
# Generate a GET URL valid for 1 hour (default)
download_url = client.presign_url("my-bucket", "private/document.pdf")

# Generate a GET URL valid for 24 hours
long_url = client.presign_url("my-bucket", "files/data.csv", expires_in=86400)

# Generate a HEAD URL for metadata access only
metadata_url = client.presign_url("my-bucket", "info.json", method="HEAD", expires_in=1800)

print(f"Share this URL: {download_url}")
# Anyone with this URL can download the file for the next hour
```

**Security Notes:**

- Pre-signed URLs contain embedded credentials and should be treated as sensitive
- URLs are only valid for the specified time period
- Anyone with the URL can access the object using the specified method
- Consider using shorter expiration times for sensitive content

### Utility Functions

#### `parse_s3_url(url)`

Parses an S3 URL into bucket and key components with automatic service detection.

**Parameters:**

- `url` (string): S3 URL (s3://, http://, or https://)

**Returns:**

- Dictionary with fields:
  - `bucket` (string): Bucket name
  - `key` (string): Object key
  - `service_type` (string): Detected service type

#### `generate_s3_url(bucket, key)`

Generates a standard S3 URL.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key

**Returns:**

- String containing S3 URL

#### `get_public_url(bucket, key, region="us-east-1", endpoint="", use_ssl=True, service_type="aws")`

Generates a public HTTP URL for an object with multi-provider support.

**Parameters:**

- `bucket` (string): Bucket name
- `key` (string): Object key
- `region` (string, optional): S3 region
- `endpoint` (string, optional): Custom endpoint
- `use_ssl` (bool, optional): Use HTTPS
- `service_type` (string, optional): Service type for URL format

**Returns:**

- String containing public URL

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

### Advanced Usage with File Operations

```python
load("s3", "create_client", "parse_s3_url", "validate_bucket_name", "get_public_url")

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

        # Get comprehensive bucket information
        bucket_info = client.get_bucket_info(bucket_name)
        print(f"Bucket created in region: {bucket_info['region']}")
        print(f"Versioning: {bucket_info['versioning_status']}")
    else:
        print(f"Bucket already exists: {bucket_name}")
else:
    print(f"Invalid bucket name: {bucket_name}")

# Upload file with comprehensive metadata
client.put_object_file(
    bucket_name,
    "documents/important.pdf",
    "/local/path/document.pdf",
    content_type="application/pdf",
    metadata={
        "author": "John Doe",
        "department": "Engineering",
        "classification": "internal",
    }
)

# Set additional object properties
client.set_object_info(
    bucket_name,
    "documents/important.pdf",
    cache_control="max-age=3600",
    content_disposition="attachment; filename=important.pdf",
    content_language="en-US",
    expires="2024-12-31T23:59:59Z",
    tags={"project": "alpha", "sensitive": "true"}
)

# Copy object with metadata changes
client.copy_object(
    bucket_name, "documents/important.pdf",
    bucket_name, "archive/important-backup.pdf",
    metadata={"archived": "true", "backup_date": "2024-01-15"}
)

# Parse and generate URLs with service detection
s3_url = "s3://my-bucket/path/to/file.txt"
parsed = parse_s3_url(s3_url)
print(f"Bucket: {parsed['bucket']}, Key: {parsed['key']}, Service: {parsed['service_type']}")

# Generate public URL for Cloudflare R2
public_url = get_public_url(
    "my-bucket", "documents/public.pdf",
    service_type="cloudflare",
    endpoint="account.r2.cloudflarestorage.com"
)
print(f"Public URL: {public_url}")

# Download file directly to filesystem
client.get_object_file(bucket_name, "documents/important.pdf", "/local/download/document.pdf")

# Generate pre-signed URLs for secure sharing
download_url = client.presign_url(bucket_name, "documents/important.pdf", expires_in=3600)
print(f"Temporary download URL (1 hour): {download_url}")

# Generate long-term pre-signed URL for public sharing  
share_url = client.presign_url(bucket_name, "documents/public-report.pdf", expires_in=604800)  # 7 days
print(f"Share this URL (valid for 7 days): {share_url}")

# Generate metadata-only access URL
metadata_url = client.presign_url(bucket_name, "documents/info.json", method="HEAD", expires_in=1800)
print(f"Metadata access URL (30 minutes): {metadata_url}")
```

### Multi-Provider URL Handling

```python
load("s3", "parse_s3_url", "get_public_url")

# Parse various provider URLs
urls = [
    "https://bucket.s3.amazonaws.com/file.txt",
    "https://bucket.nyc3.digitaloceanspaces.com/file.txt", 
    "https://account.r2.cloudflarestorage.com/bucket/file.txt",
    "https://localhost:9000/bucket/file.txt"
]

for url in urls:
    parsed = parse_s3_url(url)
    print(f"URL: {url}")
    print(f"  Service: {parsed['service_type']}")
    print(f"  Bucket: {parsed['bucket']}")
    print(f"  Key: {parsed['key']}")
    print()

# Generate URLs for different providers
providers = [
    {"service": "aws", "region": "us-west-2"},
    {"service": "digitalocean", "region": "nyc3"},
    {"service": "cloudflare", "endpoint": "account.r2.cloudflarestorage.com"},
    {"service": "minio", "endpoint": "localhost:9000", "use_ssl": False}
]

for provider in providers:
    url = get_public_url(
        "my-bucket", "data/file.txt",
        service_type=provider["service"],
        region=provider.get("region", "auto"),
        endpoint=provider.get("endpoint", ""),
        use_ssl=provider.get("use_ssl", True)
    )
    print(f"{provider['service']}: {url}")
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
        
        # Try to upload object with metadata
        client.put_object(
            "my-test-bucket", "test.txt", "Hello World",
            content_type="text/plain",
            metadata={"uploaded_by": "starlark"}
        )
        print("Object uploaded successfully")
        
        # Get object info
        info = client.get_object_info("my-test-bucket", "test.txt")
        print(f"Object size: {info['size']} bytes")
        print(f"Content type: {info['content_type']}")
        
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

### Run Test Suite

Execute the comprehensive test suite:

```bash
go test -v
```

**📋 For extensive integration testing with real cloud providers, see [test/s3/README.md](test/s3/README.md)**

The Go test suite includes:
- Client creation and configuration validation
- Utility function correctness  
- API method availability verification
- URL parsing accuracy for all supported providers
- Provider detection algorithm testing
- Service type detection and smart defaults

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📞 Support

For support, please open an issue on the GitHub repository.

---

**Made with ❤️ for the Starlark community**
