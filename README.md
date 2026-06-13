# 🗂️ `s3` — S3-compatible storage for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/s3.svg)](https://pkg.go.dev/github.com/starpkg/s3)

Universal S3-compatible storage operations for Starlark scripts. Built on the
AWS SDK for Go v2, the module works with Amazon S3, MinIO, DigitalOcean Spaces,
Cloudflare R2, Wasabi, Backblaze B2, and other S3-compatible services, with
smart provider auto-detection from endpoints, regions, and credentials.

## Installation

```bash
go get github.com/starpkg/s3
```

## Functions

These are the names registered by `LoadModule` (loadable via `load("s3", ...)`).

| Function | Signature | Description |
|----------|-----------|-------------|
| `create_client` | `create_client(**kwargs) -> Client` | Create an S3 client (see [Configuration](#configuration) for accepted keyword arguments). Credentials are **not** accepted here — they are host-injected. |
| `validate_bucket_name` | `validate_bucket_name(name) -> bool` | Report whether `name` is a valid S3 bucket name. |
| `validate_object_key` | `validate_object_key(key) -> bool` | Report whether `key` is a valid S3 object key (non-empty, ≤ 1024 bytes, no ASCII control characters `0x00`–`0x0F`). |
| `get_supported_services` | `get_supported_services() -> list[str]` | List the supported service-type strings (e.g. `"aws"`, `"minio"`, `"cloudflare"`). |

The `Client` value returned by `create_client` exposes these methods:

| Method | Signature | Description |
|--------|-----------|-------------|
| `get_client_info` | `client.get_client_info() -> struct` | Return the client's effective config: `service_type`, `region`, `endpoint`, the non-secret options, and `access_key_set` / `secret_key_set` / `session_token_set` booleans (secret values are never exposed). |
| `get_public_url` | `client.get_public_url(bucket, key) -> str` | Build a public HTTP(S) URL for an object using the client's own `region` / `endpoint` / `use_ssl` / `service_type` config. |
| `presign_url` | `client.presign_url(bucket, key, expires_in=3600, method="GET") -> str` | Generate a pre-signed URL (`method` is `"GET"` or `"HEAD"`). |
| `create_bucket` | `client.create_bucket(bucket, region=None)` | Create a bucket. |
| `delete_bucket` | `client.delete_bucket(bucket, force=False)` | Delete a bucket (`force=True` deletes its objects first). |
| `list_buckets` | `client.list_buckets() -> list[dict]` | List buckets in the account. |
| `bucket_exists` | `client.bucket_exists(bucket) -> bool` | Report whether a bucket exists. |
| `get_bucket_info` | `client.get_bucket_info(bucket) -> dict` | Comprehensive bucket information. |
| `put_object` | `client.put_object(bucket, key, content, **kwargs)` | Upload an object from a string. |
| `put_object_file` | `client.put_object_file(bucket, key, file_path, **kwargs)` | Upload an object from a local file. |
| `get_object` | `client.get_object(bucket, key) -> str` | Download an object as a string. |
| `get_object_file` | `client.get_object_file(bucket, key, file_path)` | Download an object to a local file. |
| `delete_object` | `client.delete_object(bucket, key)` | Delete an object. |
| `list_objects` | `client.list_objects(bucket, prefix="", delimiter="", max_keys=1000) -> list[dict]` | List objects in a bucket (returns the object list directly). |
| `object_exists` | `client.object_exists(bucket, key) -> bool` | Report whether an object exists. |
| `get_object_info` | `client.get_object_info(bucket, key) -> dict` | Object metadata. |
| `set_object_info` | `client.set_object_info(bucket, key, **kwargs)` | Set object metadata/properties (in-place copy). |
| `copy_object` | `client.copy_object(src_bucket, src_key, dst_bucket, dst_key, **kwargs)` | Copy an object. |

Object-writing methods (`put_object`, `put_object_file`, `set_object_info`,
`copy_object`) accept these optional keyword arguments: `content_type`,
`metadata` (dict), `tags` (dict), `cache_control`, `content_disposition`,
`content_encoding`, `content_language`, `expires` (RFC 3339 string).

## Usage

```python
load("s3", "create_client")

# Create a client — credentials come from the host (see Safety), region is
# enough for AWS auto-detection.
client = create_client(region="us-west-2")

# Create a bucket
client.create_bucket("my-bucket")

# Upload and download
client.put_object("my-bucket", "hello.txt", "Hello, World!")
content = client.get_object("my-bucket", "hello.txt")   # => "Hello, World!"

# List objects (returns a list directly)
for obj in client.list_objects("my-bucket", prefix="docs/"):
    print(obj["key"], obj["size"])

# A public URL (from the client's own config) and a temporary signed URL
public = client.get_public_url("my-bucket", "hello.txt")
signed = client.presign_url("my-bucket", "hello.txt", expires_in=3600)
```

### MinIO and other providers

```python
load("s3", "create_client")

# MinIO via an explicit endpoint
minio = create_client(service_type="minio", endpoint="localhost:9000", use_ssl=False)

# Cloudflare R2 (auto-detected from the endpoint)
r2 = create_client(endpoint="https://<account>.r2.cloudflarestorage.com")

# Inspect what a client resolved to
info = minio.get_client_info()
print(info.service_type, info.region, info.endpoint)
print("credentials present:", info.access_key_set and info.secret_key_set)
```

### Object metadata and copy

```python
client.set_object_info(
    "my-bucket", "document.pdf",
    content_type="application/pdf",
    cache_control="max-age=3600",
    metadata={"author": "Ada", "version": "1.0"},
    tags={"project": "alpha"},
)

client.copy_object(
    "src-bucket", "src/file.txt",
    "dst-bucket", "dst/file.txt",
    metadata={"copied": "true"},
)
```

## Smart provider detection

Omit `service_type` (or set it to `"auto"`) to let the client detect the
provider from the endpoint, region, or host-injected access key. Detection runs
a priority-ordered rule engine:

| Priority | Signal | Examples |
|----------|--------|----------|
| Highest | Endpoint pattern | `amazonaws.com`→AWS, `r2.cloudflarestorage.com`→R2, `digitaloceanspaces.com`→DigitalOcean, `wasabisys.com`→Wasabi, `backblazeb2.com`→Backblaze, `aliyuncs.com`→Alibaba |
| High | Special region | `region="auto"`→Cloudflare R2 |
| Medium | Access-key pattern | `AKIA…`/`ASIA…`→AWS, 32-char hex→R2 (host key, not script) |
| Lower | Region format | `us-west-2`→AWS, `nyc3`/`fra1`→DigitalOcean |
| Lowest | Endpoint shape | `localhost:9000`→MinIO, `min.io` domain→MinIO |

`get_supported_services()` returns the service-type strings the module knows.

| Service | Service type | Default region |
|---------|--------------|----------------|
| Amazon S3 | `"aws"` | `us-east-1` |
| MinIO | `"minio"` | `us-east-1` |
| DigitalOcean Spaces | `"digitalocean"` | `nyc3` |
| Linode Object Storage | `"linode"` | `us-east-1` |
| Wasabi | `"wasabi"` | `us-east-1` |
| Backblaze B2 | `"backblaze"` | `us-west-000` |
| Cloudflare R2 | `"cloudflare"` | `auto` |
| Scaleway | `"scaleway"` | `fr-par` |
| Alibaba Cloud OSS | `"alibaba"` | `cn-hangzhou` |
| Google Cloud Storage | `"google"` | `us-central1` |
| Oracle Cloud | `"oracle"` | `us-ashburn-1` |
| IBM Cloud | `"ibm"` | `us-south` |

Provider feature support varies (for example, Cloudflare R2 does not support S3
object tagging). The module degrades gracefully and surfaces clear errors for
operations a provider does not support. To branch on the provider at runtime,
read it from the client:

```python
if client.get_client_info().service_type == "aws":
    client.put_object(bucket, key, content, tags={"env": "prod"})
else:
    client.put_object(bucket, key, content)   # e.g. R2 has no tagging
```

## Safety

**Credentials are never passed from a Starlark script.** They are injected by
the Go host, so an untrusted script can use S3 without ever seeing or hardcoding
secret keys. `create_client(...)` does **not** accept `access_key` /
`secret_key` / `session_token`; passing them raises an `unexpected keyword
argument` error. Provide credentials by either:

- the environment variables `S3_ACCESS_KEY` / `S3_SECRET_KEY` /
  `S3_SESSION_TOKEN` (these back the module's secret config options), or
- the AWS default credential chain (`AWS_ACCESS_KEY_ID`, shared config/profile,
  IAM role, etc.).

A script may still choose the non-secret provider/region/endpoint, e.g.
`create_client(service_type="minio", endpoint="localhost:9000")`, and inspect
whether credentials are present with `client.get_client_info()` (which reports
`*_set` booleans, never the secret values).

`validate_object_key` rejects keys that are empty, longer than 1024 bytes, or
that contain ASCII control characters (`0x00`–`0x0F`).

## Configuration

All options are accepted as keyword arguments to `create_client(...)` and back
the module's defaults.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `service_type` | `string` | `"auto"` | S3 service type (`aws`, `minio`, `cloudflare`, …); `"auto"` enables detection |
| `access_key` | `string` | `""` | Access key ID (secret; host-injected only) |
| `secret_key` | `string` | `""` | Secret access key (secret; host-injected only) |
| `session_token` | `string` | `""` | Session token for temporary credentials (secret; host-injected only) |
| `region` | `string` | `"us-east-1"` | S3 region |
| `endpoint` | `string` | `""` | Custom S3 endpoint URL |
| `force_path_style` | `bool` | `false` | Force path-style addressing |
| `use_ssl` | `bool` | `true` | Use SSL/TLS for connections |
| `timeout` | `int` | `30` | Request timeout in seconds |
| `max_retries` | `int` | `3` | Maximum retry attempts |
| `part_size` | `int` | `5242880` | Multipart upload part size in bytes (5 MiB) |
| `concurrency` | `int` | `3` | Number of concurrent operations |
| `enable_logging` | `bool` | `false` | Enable debug logging |
| `user_agent` | `string` | `"Starlark-S3/1.0"` | Custom user agent string |

Settable via `S3_SERVICE_TYPE` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` /
`S3_SESSION_TOKEN` / `S3_REGION` / `S3_ENDPOINT` / `S3_FORCE_PATH_STYLE` /
`S3_USE_SSL` / `S3_TIMEOUT` / `S3_MAX_RETRIES` / `S3_PART_SIZE` /
`S3_CONCURRENCY` / `S3_ENABLE_LOGGING` / `S3_USER_AGENT` (the env var is
`S3_` + the option name uppercased).

## License

MIT — see [LICENSE](LICENSE).
