# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`starpkg/s3` is an **L4 domain module** of the Star\* ecosystem: it exposes
S3-compatible object storage to Starlark scripts. A script loads the module,
creates a client, and performs bucket/object operations with Go data marshalled
to and from Starlark values.

Within the starpkg philosophy — *support for necessary local operations plus
simple abstractions over common online services, for ease of use* — `s3` is
primarily an **online-service abstraction**: one uniform Starlark surface over a
family of remote object stores (Amazon S3, MinIO, DigitalOcean Spaces, Cloudflare
R2, Wasabi, Backblaze B2, Linode, Scaleway, Alibaba OSS, Google Cloud Storage,
Oracle, IBM, and any custom S3-compatible endpoint). It also touches the **local**
filesystem at two points — `put_object_file` reads a local file to upload,
`get_object_file` writes a downloaded object to disk — so it straddles the line,
but its centre of gravity is the online service.

It is built on the **AWS SDK for Go v2** (`aws-sdk-go-v2`), which speaks the S3
protocol; non-AWS providers are reached by pointing the SDK at a custom endpoint.

Layer position: depends downward on `starpkg/base` (the module/config system),
`1set/starlet` (the Machine + `dataconv` marshalling + thread context), and
transitively `1set/starlight` + `go.starlark.net`. Nothing in the ecosystem
depends on it.

## Dev commands

Pure Go library with a Makefile. From this repo:

```bash
make test                                  # -race -cover, the working bar
make ci                                    # -race -cover profile + bench compile (what CI runs)
go test ./... -run TestDetectServiceType   # a single test
make bench                                 # benchmarks
gofmt -l . && go vet ./...                 # must be clean before commit
```

**Verify on the go floor in Docker** — this repo's floor is **go 1.21** (see
Release discipline), and the local toolchain is newer. Behavior on the floor must
be checked in a container:

```bash
docker run --rm -v "$PWD":/src -v "$HOME/go/pkg/mod":/go/pkg/mod -w /src golang:1.21 go test -race -count=1 ./...
```

Unit tests are hermetic: detection/validation/config and the client-construction
API run offline with no network or credentials. The live integration scripts in
`TestStarlarkScripts` self-skip unless `S3_RUN_INTEGRATION=1` is set (with real
`S3_ACCESS_KEY` / `S3_SECRET_KEY`); never commit credentials. Those scripts live
under `../test/s3/*.star` in the **private `starpkg/test` repo** and auto-skip
when that directory is absent (e.g. in CI).

Documentation gate: `go run github.com/1set/meta/doccov@master .` must exit 0 —
every `starlark.NewBuiltin` name has to appear as a backtick word in `README.md`.

## Architecture (the part that spans files)

The module is a **one-client, many-providers bridge**: a single `create_client`
builds an AWS-SDK S3 client (optionally aimed at a custom endpoint), and the
returned object exposes ~18 storage methods regardless of the backing provider.

- **`s3.go`** — the module entry and the entire script-facing surface. `Module`
  wraps a `base.ConfigurableModule`; `NewModule()` constructs it with the 14
  config options. `LoadModule()` registers four module builtins — `create_client`,
  `validate_bucket_name`, `validate_object_key`, `get_supported_services`.
  `ClientWrapper` is the returned Starlark value (implements `starlark.Value` +
  `HasAttrs`); its `methodMap` registers each client method as a `NewBuiltin` and
  the per-method `star*`/`<verb>` Go functions live here.
- **`config.go`** — config-key constants, `ClientConfig` (the resolved client
  settings) with `Validate`/`detectServiceType`, `ObjectOptions` and
  `ListObjectsOptions` (the optional-argument carriers + their `ApplyTo…` mappers
  onto AWS SDK inputs), and the `DetectionRule` type plus its predicate helpers
  (`hasEndpointContaining`, `hasRegionPattern`, `hasAccessKeyPattern`, …).
- **`client.go`** — `Client`, the thin wrapper over `*s3.Client`. `NewClient` /
  `createAWSConfig` build the SDK client (static credentials only when both
  access+secret are present, else the AWS default chain). Every storage verb
  (`CreateBucket`, `ListBuckets`, `PutObject`, `GetObject`, `CopyObject`,
  `PresignURL`, …) lives here, along with the result types `BucketInfo`,
  `ObjectInfo`, `ListObjectsResult` and their `MarshalStarlark` methods.
- **`provider.go`** — the provider registry: `ProviderConfig` per service,
  `providerConfigs`/`providerOrder`, `GetAllProviders`, `DetectProviderFromConfig`
  (priority-ordered rule engine), and `GenerateURLWithProvider` (public-URL
  construction).
- **`utils.go`** — input validation (`validateBucketName`, `validateObjectKey`),
  Starlark↔Go conversion helpers, and `parseObjectOptions` (the single place that
  turns the object-writing keyword args into an `ObjectOptions`).

Data flow: `create_client` → resolve config (module defaults overridden by
non-secret script kwargs) → `detectServiceType` if `auto` → `NewClient` →
`ClientWrapper`. Each method unpacks args, calls the matching `Client` verb with
the thread context (`dataconv.GetThreadContext`), and marshals the result back
via `MarshalStarlark` / `dataconv.Marshal`.

## Invariants / hardening (preserve when editing)

1. **Credentials are host-injected, never script-passed (PKG-15).**
   `create_client` deliberately does **not** accept `access_key` / `secret_key` /
   `session_token` — they come only from the module's secret config options, the
   `S3_*` environment variables, or the AWS default credential chain. Passing them
   as script kwargs is rejected by `UnpackArgs` as an unexpected keyword argument.
   Do not add credential parameters to the script surface.
2. **Secrets are never echoed.** `get_client_info` reports only `*_set` booleans
   (`access_key_set`, …), never the secret values. Keep it that way.
3. **No host panics from script input.** Arg unpacking, validation, and
   marshalling return Starlark-level errors; nothing reaches a `panic`.
4. **Graceful provider degradation.** `get_bucket_info` / `get_object_info` make
   several best-effort AWS calls (versioning, encryption, CORS, tags); a provider
   that lacks a feature must not fail the whole call — missing data is simply
   omitted, not an error.
5. **Backward compatibility.** `NewModule()` and the default config values are the
   historical behavior; any new lever must default to it so existing scripts run
   identically.
6. **Confined local file access (PKG-15).** `put_object_file` / `get_object_file`
   are the only two operations that touch the local filesystem. Their
   script-supplied `file_path` is routed through `ClientWrapper.resolveFilePath`
   (`util.ResolveUnder`) **before** it reaches `os.Open` / `os.Create`, so a script
   cannot read or write arbitrary host files — every path is confined under the
   **host-only** `file_root` (empty = working directory), a `..`/symlink escape is
   rejected, and an "absolute" path is re-anchored under the root. The confinement
   is disabled only by the host-only `allow_unsafe_file_paths`. Keep both file ops
   calling `resolveFilePath` first; both levers are host-only so a script can't
   widen its own reach. The root is **snapshotted absolute at module construction**
   (`NewModule` → `absFileRoot`, in Go before any script runs), so a script that
   changes the process working directory (e.g. a `path.chdir` builtin) — whether
   before or after `create_client` — cannot move the jail. If the root can't be
   made absolute, `absFileRoot` returns `""` and `resolveFilePath` **fails closed**
   (rejects every path) rather than falling back to a movable working-dir root.
   Residuals (documented, not gaps in this module): `util.ResolveUnder` resolves
   symlinks at check time while `os.Open`/`os.Create` follow them at use time, so a
   root the host lets an attacker write to is TOCTOU-swappable — the host must own
   the `file_root` tree.
7. **Bounded in-memory read.** `get_object` reads the whole object into a Starlark
   string, so it routes through `util.ReadAllLimited(reader, maxObjectSize)` — the
   host-only `max_object_size` (default 256 MiB, `0` = unlimited) caps it so a huge
   object can't exhaust host memory. (`get_object_file` streams to disk via
   `io.Copy`, so it is not an in-memory concern; its path is jailed by invariant 6.)

## Test organization

Group by functional goal — **do not add one `*_test.go` per fix.** Two thematic
files are the home:

- **`detection_test.go`** — Go-level unit tests for provider detection and config
  validation (credentials are host config, so access-key-pattern detection is
  tested here with `ClientConfig` values, not from scripts).
- **`example_test.go`** — the Starlark-driven API tests (`TestS3Module`,
  `TestS3ClientInfoAndPublicURL`, `TestS3*Operations`, `TestS3ValidationFunctions`,
  `TestS3PresignURL`, …) plus `TestStarlarkScripts`, the opt-in `../test/s3`
  integration harness. Add a new test as a **section here**, not a new file.

Tests are table/example-driven; no third-party test framework. Network-dependent
cases self-skip when creds/endpoints are absent.

## Documentation

Three layers must stay in sync (enforced by the doc standard,
`plan/starpkg文档标准（DOC-STD）`):

- **`README.md`** — every script-facing builtin and client method documented as a
  backtick whole-word (the `doccov` gate fails on omission); the *Configuration*
  and *Safety* sections cover the host-side config keys, `S3_*` env vars, and the
  host-injected-credentials rule. Function names/signatures must match the code.
- **GoDoc** — package comment + a doc comment on every exported symbol whose first
  word is the symbol name (gated by `revive`'s `exported` rule in CI).
- **CLAUDE.md** — this file.

## Release discipline

- **Floor = go 1.21**, raised under an ENG-SEP because the pinned AWS SDK for Go
  v2 requires it; the floor only rises in its own dedicated PR.
- **CI matrix** = `[1.21.x, 1.25.x]` via the centralized reusable workflow in
  `1set/meta` (`.github/workflows/build.yml`, pinned to a commit SHA;
  `doc-coverage: true` runs the `doccov` gate).
- **Bumping the version, the go floor, or tagging are user-confirmed actions** —
  never tag autonomously; default to patch bumps; a published tag/release is
  immutable in the Go module proxy, so it is never deleted or re-pointed.
