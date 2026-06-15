# 🗂️ `s3` — S3-compatible storage for Starlark

[![Go Reference](https://pkg.go.dev/badge/github.com/starpkg/s3.svg)](https://pkg.go.dev/github.com/starpkg/s3)
[![license](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![codecov](https://codecov.io/gh/starpkg/s3/graph/badge.svg)](https://codecov.io/gh/starpkg/s3)
![binary footprint](https://img.shields.io/badge/binary_footprint-%2B7.5_MB-blue)

Universal S3-compatible storage operations for Starlark scripts. Built on the
AWS SDK for Go v2, the module works with Amazon S3, MinIO, DigitalOcean Spaces,
Cloudflare R2, Wasabi, Backblaze B2, and other S3-compatible services, with
smart provider auto-detection from endpoints, regions, and credentials.

## Overview

Within the starpkg philosophy — *support for necessary local operations plus
simple abstractions over common online services, for ease of use* — `s3` is an
**online-service abstraction**: it puts a small, uniform Starlark surface over a
family of remote object stores. It also touches the **local** filesystem at two
points (`put_object_file` reads a local file to upload, `get_object_file` writes
a downloaded object to a local file), so it straddles the line, but its centre
of gravity is the online service.

- **One client, many providers** — `create_client` builds a client for Amazon
  S3, MinIO, DigitalOcean Spaces, Cloudflare R2, Wasabi, Backblaze B2, and more,
  with smart auto-detection from the endpoint, region, or credentials.
- **Buckets and objects** — create / delete / list / inspect buckets; put / get
  / copy / delete / list objects, with metadata, tags, and content headers.
- **Local-file helpers** — `put_object_file` / `get_object_file` upload from and
  download to the local filesystem.
- **URLs** — public URLs from the client's config (`get_public_url`) and
  temporary pre-signed URLs (`presign_url`).
- **Credentials are host-injected, never script-passed** — a script chooses the
  provider / region / endpoint but never sees secret keys.

For the complete per-builtin reference — signatures, parameters, returns,
errors, examples — and the configuration accessors, see
**[docs/API.md](docs/API.md)**.

## Installation

```bash
go get github.com/starpkg/s3
```

## Quickstart

```python
load("s3", "create_client")

# Create a client — credentials come from the host, region is enough for AWS
# auto-detection.
client = create_client(region="us-west-2")

# Upload and download
client.put_object("my-bucket", "hello.txt", "Hello, World!")
content = client.get_object("my-bucket", "hello.txt")   # => "Hello, World!"

# List objects (returns the object list directly)
for obj in client.list_objects("my-bucket", prefix="docs/"):
    print(obj["key"], obj["size"])

# A public URL (from the client's own config) and a temporary signed URL
public = client.get_public_url("my-bucket", "hello.txt")
signed = client.presign_url("my-bucket", "hello.txt", expires_in=3600)
```

```python
load("s3", "create_client")

# MinIO via an explicit endpoint; inspect what the client resolved to.
minio = create_client(service_type="minio", endpoint="localhost:9000", use_ssl=False)
info = minio.get_client_info()
print(info.service_type, info.region, info.endpoint)
print("credentials present:", info.access_key_set and info.secret_key_set)
```

## Starlark API at a glance

Top-level builtins (`load("s3", …)`):

- `create_client(service_type?, region?, endpoint?, force_path_style?, use_ssl?, timeout?, max_retries?, part_size?, concurrency?, enable_logging?, user_agent?)` — build an S3 client (credentials are host-injected, not accepted here).
- `validate_bucket_name(name)` — whether `name` is a valid S3 bucket name.
- `validate_object_key(key)` — whether `key` is a valid S3 object key.
- `get_supported_services()` — the list of supported service-type strings.

`Client` object methods (returned by `create_client`):

- `get_client_info()` — effective config struct (secret values reported only as `*_set` booleans).
- `get_public_url(bucket, key)` — build a public URL from the client's config.
- `presign_url(bucket, key, expires_in=3600, method="GET")` — temporary signed URL (`GET`/`PUT`/`HEAD`).
- `create_bucket(bucket, region=None)` — create a bucket.
- `delete_bucket(bucket, force=False)` — delete a bucket (`force=True` empties it first).
- `list_buckets()` — list buckets as a list of dicts.
- `bucket_exists(bucket)` — whether a bucket exists.
- `get_bucket_info(bucket)` — comprehensive bucket info dict.
- `put_object(bucket, key, content, **options)` — upload an object from a string.
- `put_object_file(bucket, key, file_path, **options)` — upload an object from a local file.
- `get_object(bucket, key)` — download an object as a string.
- `get_object_file(bucket, key, file_path)` — download an object to a local file.
- `delete_object(bucket, key)` — delete an object.
- `list_objects(bucket, prefix="", delimiter="", max_keys=1000)` — list objects (returns the list directly).
- `object_exists(bucket, key)` — whether an object exists.
- `get_object_info(bucket, key)` — object metadata dict.
- `set_object_info(bucket, key, **options)` — set object metadata/properties in place.
- `copy_object(src_bucket, src_key, dst_bucket, dst_key, **options)` — copy an object.

The object-writing methods (`put_object`, `put_object_file`, `set_object_info`,
`copy_object`) accept the optional keyword arguments `content_type`, `metadata`,
`tags`, `cache_control`, `content_disposition`, `content_encoding`,
`content_language`, and `expires`.

See **[docs/API.md](docs/API.md)** for the full signatures, return values,
errors, and examples of every builtin and method above.

## Configuration

The module's options (`service_type`, `region`, `endpoint`, `force_path_style`,
`use_ssl`, `timeout`, `max_retries`, `part_size`, `concurrency`,
`enable_logging`, `user_agent`) are configured via environment variables
(`S3_*`) or per-option `get_<key>` / `set_<key>` accessor builtins, and the
non-secret ones double as `create_client` defaults. Credentials (`access_key`,
`secret_key`, `session_token`) are **secret and host-injected only** — set-only,
never readable, and never `create_client` keyword arguments. See the
[Configuration section of docs/API.md](docs/API.md#configuration) for the full
option table, accessors, env vars, defaults, and the host-injected-credentials
rule.

## License

MIT — see [LICENSE](LICENSE).
