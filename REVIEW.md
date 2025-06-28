You're a Go architect and Starlark expert, and helping to complete a Starlark module in Go named "s3" for S3 (Amazon Simple Storage Service) Compatible Storage operations.

The module should provide a simple and intuitive interface for interacting with S3-compatible storage services, including operations like uploading, downloading, listing, deleting objects, managing buckets, and handling metadata. The module should be designed to be easy to use and integrate with existing Starlark code. You can refer to the AWS SDK for Go S3 client for inspiration, but the implementation should be idiomatic Go and follow best practices, also with high performance. Your new `s3` package should support various S3-compatible services like Amazon S3, MinIO, DigitalOcean Spaces, and others that implement the S3 API.

Your colleague has written down a full plan in a markdown file named `PLAN.md` in the root directory of the repository, and it should include enough information for another developer to understand and implement the module.

After reviewing this documentation, you have the following suggestions to improve the documentation:

1. Functions need to be documented, including their purpose, parameters and return values, and any other relevant information.

2. The following services are S3-compatible storage services, you should add them to the list of supported services, and show the support status for each feature, also how to use them in the module (via different parameters or options).

    Amazon S3
    Cloudflare R2
    Backblaze B2
    DigitalOcean Spaces
    MinIO

3. Azure Storage Blob is not S3-compatible, you should not mention it in the documentation, and you should not support it.

4. `connect` function should be renamed to `create_client` to be consistent with the `web` module.

5. For complicated examples, you should put them in separate *.star files as attachment under folder `examples`.

Please modify the `PLAN.md` file to follow these suggestions to make this plan more clear and complete.
