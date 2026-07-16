# `s3` — Starlark API Reference

The complete reference for every script-facing builtin, client method, and
configuration accessor exposed by the `s3` module. For an overview,
installation, and a quickstart, see the [README](../README.md).

The module exposes four top-level builtins via `load("s3", …)` —
`create_client`, `validate_bucket_name`, `validate_object_key`,
`get_supported_services` — plus a set of configuration accessors
(`get_<key>` / `set_<key>`) generated from the module's options. A
`create_client(...)` call returns a `Client` object that carries the bucket and
object methods (`put_object`, `get_object`, `list_objects`, `presign_url`, …).

> **Credentials are never passed from a script.** `create_client` does **not**
> accept `access_key` / `secret_key` / `session_token`; they are host-injected
> only (see [Configuration](#configuration)). Passing them as keyword arguments
> raises an `unexpected keyword argument` error.

## Contents

- [Module functions](#module-functions)
- [Client object](#client-object)
- [Client information](#client-information)
- [URL generation](#url-generation)
- [Bucket operations](#bucket-operations)
- [Object operations](#object-operations)
- [Object-writing options](#object-writing-options)
- [Provider detection](#provider-detection)
- [Configuration](#configuration)

## Module functions

These are the names registered by `LoadModule`, loadable via
`load("s3", …)`.

### `create_client(service_type?, region?, endpoint?, force_path_style?, use_ssl?, timeout?, max_retries?, part_size?, concurrency?, enable_logging?, user_agent?)`

Creates an S3 client. Every parameter is optional; an argument omitted falls
back to the corresponding module config option (see
[Configuration](#configuration)). If `service_type` is `"auto"` or empty, the
provider is auto-detected (see [Provider detection](#provider-detection)).

**Credentials are not accepted here.** `access_key`, `secret_key`, and
`session_token` are host-injected only; passing them raises an `unexpected
keyword argument` error.

**Parameters:**

- `service_type` (string): S3 service type (`"aws"`, `"minio"`, `"cloudflare"`,
  …); `"auto"` (the default) enables provider detection.
- `region` (string): S3 region.
- `endpoint` (string): Custom S3 endpoint URL.
- `force_path_style` (bool): Force path-style addressing.
- `use_ssl` (bool): Use SSL/TLS for connections.
- `timeout` (int): Per-request timeout in seconds; bounds each HTTP request to
  the service.
- `max_retries` (int): Maximum attempts per request, **including the first try**
  (so `3` means up to two retries), default `3`. Passing `0` here means "use the
  configured default" (like any unset numeric argument), so it resolves to `3` —
  it does **not** mean zero retries. The effective value is always applied and
  takes precedence over an ambient `AWS_MAX_ATTEMPTS`. To instead defer to the
  SDK/env default, set the module option itself to `0` (`S3_MAX_RETRIES=0` or
  `set_max_retries(0)`), which is left unset on the client.
- `part_size` (int): Multipart upload part size in bytes.
- `concurrency` (int): Number of concurrent operations.
- `enable_logging` (bool): Enable SDK request/response logging.
- `user_agent` (string): Custom user-agent token appended to the SDK user agent.

**Returns:** A `Client` object (see [Client object](#client-object)).

**Errors:** Fails on an invalid configuration, or if the underlying AWS SDK
client cannot be constructed.

**Example:**

```python
load("s3", "create_client")

# Region is enough for AWS auto-detection; credentials come from the host.
client = create_client(region="us-west-2")

# MinIO via an explicit endpoint
minio = create_client(service_type="minio", endpoint="localhost:9000", use_ssl=False)

# Cloudflare R2 (auto-detected from the endpoint)
r2 = create_client(endpoint="https://<account>.r2.cloudflarestorage.com")
```

### `validate_bucket_name(name)`

Reports whether `name` is a valid S3 bucket name. A valid name is 3–63
characters, starts and ends with a lowercase letter or number, contains only
lowercase letters, numbers, dots, and hyphens, has no consecutive dots, is not
formatted as an IP address, and does not start with `xn--` or end with
`-s3alias`.

**Parameters:**

- `name` (string): Bucket name to validate.

**Returns:** `bool` — `True` if the name is valid, else `False`.

**Example:**

```python
load("s3", "validate_bucket_name")

print(validate_bucket_name("my-bucket"))   # True
print(validate_bucket_name("My_Bucket"))   # False (uppercase + underscore)
```

### `validate_object_key(key)`

Reports whether `key` is a valid S3 object key. A valid key is non-empty, at
most 1024 bytes long, and contains no ASCII control characters in the range
`0x00`–`0x0F`.

**Parameters:**

- `key` (string): Object key to validate.

**Returns:** `bool` — `True` if the key is valid, else `False`.

**Example:**

```python
load("s3", "validate_object_key")

print(validate_object_key("docs/report.pdf"))   # True
print(validate_object_key(""))                   # False (empty)
```

### `get_supported_services()`

Lists the supported service-type strings the module knows.

**Parameters:** None

**Returns:** `list[str]` of service-type strings (e.g. `"aws"`, `"minio"`,
`"cloudflare"`).

**Example:**

```python
load("s3", "get_supported_services")

for service in get_supported_services():
    print(service)
```

## Client object

The value returned by `create_client` is a `Client`. It is immutable, truthy,
and unhashable (cannot be used as a dict key). It exposes the methods below via
attribute access (e.g. `client.put_object(...)`).

## Client information

### `client.get_client_info()`

Returns the client's effective configuration. Secret values are never
exposed — only `*_set` booleans report whether each credential is present.

**Parameters:** None

**Returns:** A struct with the fields: `service_type`, `region`, `endpoint`,
`force_path_style`, `use_ssl`, `timeout`, `max_retries`, `part_size`,
`concurrency`, `enable_logging`, `user_agent`, plus `access_key_set`,
`secret_key_set`, and `session_token_set` booleans.

**Example:**

```python
info = client.get_client_info()
print(info.service_type, info.region, info.endpoint)
print("credentials present:", info.access_key_set and info.secret_key_set)
```

## URL generation

### `client.get_public_url(bucket, key)`

Builds a public HTTP(S) URL for an object using the client's own `region` /
`endpoint` / `use_ssl` / `service_type` configuration. (No request is made; the
URL is constructed from config.)

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.

**Returns:** `str` — the public URL.

**Example:**

```python
public = client.get_public_url("my-bucket", "hello.txt")
```

### `client.presign_url(bucket, key, expires_in=3600, method="GET")`

Generates a pre-signed URL valid for `expires_in` seconds.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.
- `expires_in` (int): Validity in seconds (default: `3600`).
- `method` (string): `"GET"`, `"PUT"`, or `"HEAD"` (case-insensitive; default:
  `"GET"`).

**Returns:** `str` — the pre-signed URL.

**Errors:** Fails if `method` is anything other than `GET`, `PUT`, or `HEAD`,
or if presigning fails.

**Example:**

```python
signed = client.presign_url("my-bucket", "hello.txt", expires_in=3600)
upload = client.presign_url("my-bucket", "upload.bin", method="PUT")
```

## Bucket operations

### `client.create_bucket(bucket, region=None)`

Creates a bucket. When `region` is omitted, the client's configured region is
used.

**Parameters:**

- `bucket` (string): Bucket name.
- `region` (string, optional): Region in which to create the bucket.

**Returns:** None

**Example:**

```python
client.create_bucket("my-bucket")
client.create_bucket("eu-bucket", region="eu-west-1")
```

### `client.delete_bucket(bucket, force=False)`

Deletes a bucket. With `force=True`, all of the bucket's objects are deleted
first. If any object cannot be deleted (object lock, governance, permissions),
the call fails with an error naming the first failure rather than reporting a
false success — the bucket is left intact so the condition is not silently
swallowed.

**Parameters:**

- `bucket` (string): Bucket name.
- `force` (bool): If `True`, delete the bucket's objects before deleting the
  bucket (default: `False`).

**Returns:** None

**Example:**

```python
client.delete_bucket("temp-bucket", force=True)
```

### `client.list_buckets()`

Lists the buckets in the account.

**Parameters:** None

**Returns:** `list[dict]` — one dict per bucket (the bucket-info fields below).

**Example:**

```python
for bucket in client.list_buckets():
    print(bucket["name"], bucket["creation_date"])
```

### `client.bucket_exists(bucket)`

Reports whether a bucket exists.

**Parameters:**

- `bucket` (string): Bucket name.

**Returns:** `bool`

**Example:**

```python
if client.bucket_exists("my-bucket"):
    print("exists")
```

### `client.get_bucket_info(bucket)`

Returns comprehensive bucket information. Several fields are gathered with
best-effort calls (versioning, encryption, CORS, policy, tags); a provider that
lacks a feature simply omits the corresponding data rather than failing.

**Parameters:**

- `bucket` (string): Bucket name.

**Returns:** `dict` with the fields `name`, `creation_date`, `region`,
`location`, `versioning_status`, `public_access_blocked`, `has_policy`,
`has_cors`, `encryption_enabled`, `encryption_type`, `object_count`,
`total_size`, `storage_class`, `tags`, `owner`, `bucket_type`.

**Example:**

```python
info = client.get_bucket_info("my-bucket")
print(info["versioning_status"], info["encryption_enabled"])
```

## Object operations

### `client.put_object(bucket, key, content, **options)`

Uploads an object from an in-memory string.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.
- `content` (string): Object body.
- `**options`: Object-writing options (see
  [Object-writing options](#object-writing-options)).

**Returns:** None

**Example:**

```python
client.put_object("my-bucket", "hello.txt", "Hello, World!")
client.put_object("my-bucket", "report.json", body, content_type="application/json")
```

### `client.put_object_file(bucket, key, file_path, **options)`

Uploads an object by reading a **local** file. (One of the module's two
local-filesystem touch points.)

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.
- `file_path` (string): Path to the local file to upload.
- `**options`: Object-writing options (see
  [Object-writing options](#object-writing-options)).

**Returns:** None

**Example:**

```python
client.put_object_file("my-bucket", "images/logo.png", "/tmp/logo.png",
                        content_type="image/png")
```

### `client.get_object(bucket, key)`

Downloads an object and returns its body as a string.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.

**Returns:** `str` — the object body.

**Example:**

```python
content = client.get_object("my-bucket", "hello.txt")   # => "Hello, World!"
```

### `client.get_object_file(bucket, key, file_path)`

Downloads an object and writes it to a **local** file. (The module's second
local-filesystem touch point.)

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.
- `file_path` (string): Destination path for the downloaded object.

**Returns:** None

**Example:**

```python
client.get_object_file("my-bucket", "images/logo.png", "/tmp/logo.png")
```

### `client.delete_object(bucket, key)`

Deletes an object.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.

**Returns:** None

**Example:**

```python
client.delete_object("my-bucket", "obsolete.txt")
```

### `client.list_objects(bucket, prefix="", delimiter="", max_keys=1000)`

Lists objects in a bucket. The object list is returned directly (not wrapped in
a result envelope).

**Parameters:**

- `bucket` (string): Bucket name.
- `prefix` (string): Only return keys beginning with this prefix (default: all).
- `delimiter` (string): Group keys by this delimiter (default: none).
- `max_keys` (int): Maximum number of items to return **in total** (default:
  `1000`). S3 returns at most 1000 items per request, so a larger `max_keys` is
  satisfied by transparently paginating and concatenating pages until the total
  is reached — you are not silently capped at one page. Memory use is bounded by
  `max_keys`, so raise it deliberately for very large buckets. When a `delimiter`
  is set, grouped common prefixes count toward this total alongside objects (S3
  counts them together).

**Returns:** `list[dict]` — one dict per object (the object-info fields below).

> **Note:** the flat list carries objects only. When `delimiter` is set, the
> grouped `common_prefixes` (the pseudo-folders) are **not** returned by this
> method; list without a delimiter to enumerate the keys themselves.

**Example:**

```python
for obj in client.list_objects("my-bucket", prefix="docs/"):
    print(obj["key"], obj["size"])
```

### `client.object_exists(bucket, key)`

Reports whether an object exists.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.

**Returns:** `bool`

**Example:**

```python
if client.object_exists("my-bucket", "hello.txt"):
    print("present")
```

### `client.get_object_info(bucket, key)`

Returns object metadata.

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.

**Returns:** `dict` with the fields `key`, `size`, `last_modified`, `etag`,
`content_type`, `content_encoding`, `content_disposition`, `content_language`,
`cache_control`, `expires`, `storage_class`, `checksum_algorithm`, `version_id`,
`is_latest`, `owner`, `metadata`, `tags`. For `list_objects` entries
`checksum_algorithm` carries the object's checksum algorithm (e.g. `"SHA256"`)
when present; `version_id` there is empty because a plain object listing carries
no version — it is populated only where the API returns one (e.g.
`get_object_info`).

**Example:**

```python
info = client.get_object_info("my-bucket", "hello.txt")
print(info["size"], info["content_type"], info["etag"])
```

### `client.set_object_info(bucket, key, **options)`

Sets object metadata and properties in place (implemented as a self-copy with
the metadata directive set to replace).

**Parameters:**

- `bucket` (string): Bucket name.
- `key` (string): Object key.
- `**options`: Object-writing options (see
  [Object-writing options](#object-writing-options)).

**Returns:** None

**Example:**

```python
client.set_object_info(
    "my-bucket", "document.pdf",
    content_type="application/pdf",
    cache_control="max-age=3600",
    metadata={"author": "Ada", "version": "1.0"},
    tags={"project": "alpha"},
)
```

### `client.copy_object(src_bucket, src_key, dst_bucket, dst_key, **options)`

Copies an object from one location to another. When any object-writing option is
supplied, the metadata directive is set to replace.

**Parameters:**

- `src_bucket` (string): Source bucket name.
- `src_key` (string): Source object key.
- `dst_bucket` (string): Destination bucket name.
- `dst_key` (string): Destination object key.
- `**options`: Object-writing options (see
  [Object-writing options](#object-writing-options)).

**Returns:** None

**Example:**

```python
client.copy_object(
    "src-bucket", "src/file.txt",
    "dst-bucket", "dst/file.txt",
    metadata={"copied": "true"},
)
```

## Object-writing options

The object-writing methods — `put_object`, `put_object_file`, `set_object_info`,
and `copy_object` — accept these optional keyword arguments:

| Option | Type | Description |
|--------|------|-------------|
| `content_type` | string | MIME type of the object |
| `metadata` | dict | User-defined metadata (string keys and values) |
| `tags` | dict | Object tags (string keys and values) |
| `cache_control` | string | `Cache-Control` header |
| `content_disposition` | string | `Content-Disposition` header |
| `content_encoding` | string | `Content-Encoding` header |
| `content_language` | string | `Content-Language` header |
| `expires` | string | Expiry timestamp as an RFC 3339 string |

Provider feature support varies (for example, Cloudflare R2 does not support S3
object tagging). The module degrades gracefully and surfaces clear errors for
operations a provider does not support; branch on
`client.get_client_info().service_type` when needed.

## Provider detection

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

`get_supported_services()` returns the service-type strings the module knows:

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

```python
if client.get_client_info().service_type == "aws":
    client.put_object(bucket, key, content, tags={"env": "prod"})
else:
    client.put_object(bucket, key, content)   # e.g. R2 has no tagging
```

## Configuration

Each module configuration option is exposed to scripts as generated accessor
builtins (loaded from the `s3` module alongside the functions above):

- **`get_<key>()`** — returns the current value of the option.
- **`set_<key>(value)`** — sets the option (returns `None`).

An option's value resolves in priority order: an explicit `set_<key>` value, the
environment variable (`S3_<KEY>`), then the default. The **non-secret** options
are also accepted as keyword arguments to `create_client(...)` (a
script-supplied value overrides the module default).

**Secret options are set-only.** The three credential options — `access_key`,
`secret_key`, `session_token` — are secret: they expose **only** a `set_<key>`
accessor and **no getter** (so a secret value can never be read back from a
script). They are also **not** accepted as `create_client(...)` keyword
arguments (passing them raises an `unexpected keyword argument` error). Provide
credentials via the `S3_ACCESS_KEY` / `S3_SECRET_KEY` / `S3_SESSION_TOKEN`
environment variables, the host config, or the AWS default credential chain.
`client.get_client_info()` reports only `access_key_set` / `secret_key_set` /
`session_token_set` booleans, never the values.

**Host-only file-access policy.** `file_root` and `allow_unsafe_file_paths` govern
which local paths `put_object_file` / `get_object_file` may touch. They are
**host-only**: only their getters (`get_file_root`, `get_allow_unsafe_file_paths`)
are generated — **`set_file_root` and `set_allow_unsafe_file_paths` are
intentionally NOT exposed**, so a script cannot widen its own filesystem reach.
By default those two methods confine every local path under `file_root` (empty =
the process working directory), rejecting any path that escapes it via `..` or a
symlink; an "absolute" path is re-anchored under the root rather than reaching the
real host path. `allow_unsafe_file_paths=true` is the host's explicit opt-out that
disables the confinement.

**`max_object_size`** is a third host-only lever: it bounds how many bytes
`get_object` reads into memory, so a hostile or oversized object can't exhaust
host memory. Only `get_max_object_size` is generated (no `set_max_object_size`);
`0` disables the limit and the default is 256 MiB; a negative value is a
misconfiguration and falls back to the default (fail-safe, never fail-open).

| Option | Getter | Setter | Type | Env var | Default | Description |
|--------|--------|--------|------|---------|---------|-------------|
| `service_type` | `get_service_type` | `set_service_type` | string | `S3_SERVICE_TYPE` | `"auto"` | S3 service type (`aws`, `minio`, `cloudflare`, …); `"auto"` enables detection |
| `access_key` | *(secret — none)* | `set_access_key` | string | `S3_ACCESS_KEY` | `""` | Access key ID (secret; host-injected, set-only, not a `create_client` kwarg) |
| `secret_key` | *(secret — none)* | `set_secret_key` | string | `S3_SECRET_KEY` | `""` | Secret access key (secret; host-injected, set-only, not a `create_client` kwarg) |
| `session_token` | *(secret — none)* | `set_session_token` | string | `S3_SESSION_TOKEN` | `""` | Session token for temporary credentials (secret; host-injected, set-only, not a `create_client` kwarg) |
| `region` | `get_region` | `set_region` | string | `S3_REGION` | `"us-east-1"` | S3 region |
| `endpoint` | `get_endpoint` | `set_endpoint` | string | `S3_ENDPOINT` | `""` | Custom S3 endpoint URL |
| `force_path_style` | `get_force_path_style` | `set_force_path_style` | bool | `S3_FORCE_PATH_STYLE` | `false` | Force path-style addressing |
| `use_ssl` | `get_use_ssl` | `set_use_ssl` | bool | `S3_USE_SSL` | `true` | Use SSL/TLS for connections |
| `timeout` | `get_timeout` | `set_timeout` | int | `S3_TIMEOUT` | `30` | Per-request timeout in seconds |
| `max_retries` | `get_max_retries` | `set_max_retries` | int | `S3_MAX_RETRIES` | `3` | Maximum attempts per request, incl. the first try |
| `part_size` | `get_part_size` | `set_part_size` | int | `S3_PART_SIZE` | `5242880` | Multipart upload part size in bytes (5 MiB) |
| `concurrency` | `get_concurrency` | `set_concurrency` | int | `S3_CONCURRENCY` | `3` | Number of concurrent operations |
| `enable_logging` | `get_enable_logging` | `set_enable_logging` | bool | `S3_ENABLE_LOGGING` | `false` | Enable debug logging |
| `user_agent` | `get_user_agent` | `set_user_agent` | string | `S3_USER_AGENT` | `"Starlark-S3/1.0"` | Custom user agent string |
| `file_root` | `get_file_root` | `set_file_root` — **host-only, not generated** | string | `S3_FILE_ROOT` | `""` | Root that `put_object_file`/`get_object_file` paths are confined under (`""` = working directory) |
| `allow_unsafe_file_paths` | `get_allow_unsafe_file_paths` | `set_allow_unsafe_file_paths` — **host-only, not generated** | bool | `S3_ALLOW_UNSAFE_FILE_PATHS` | `false` | Disable the `file_root` confinement (host opt-out) |
| `max_object_size` | `get_max_object_size` | `set_max_object_size` — **host-only, not generated** | int | `S3_MAX_OBJECT_SIZE` | `268435456` | Max bytes `get_object` reads into memory (256 MiB; `0` = unlimited) |

The env var for any option is `S3_` + the option name uppercased.

**Example:**

```python
load(
    "s3",
    "create_client",
    # non-secret getters
    "get_service_type", "get_region", "get_endpoint", "get_force_path_style",
    "get_use_ssl", "get_timeout", "get_max_retries", "get_part_size",
    "get_concurrency", "get_enable_logging", "get_user_agent",
    # non-secret setters
    "set_service_type", "set_region", "set_endpoint", "set_force_path_style",
    "set_use_ssl", "set_timeout", "set_max_retries", "set_part_size",
    "set_concurrency", "set_enable_logging", "set_user_agent",
    # secret setters (no getters)
    "set_access_key", "set_secret_key", "set_session_token",
)

set_region("eu-west-1")
print(get_region())          # "eu-west-1"

client = create_client()     # uses region=eu-west-1
```
