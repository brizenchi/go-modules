# Changelog — foundation/ossx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2026

### Added

- Initial release. `Bucket` port with `Put / Get / Stat / Delete /
  PresignGet / PresignPut`.
- `PutOptions` (ContentType, CacheControl, ContentDisposition,
  Metadata, ACL), `PresignPutOptions`, `ObjectInfo`.
- Sentinel errors `ErrNotFound`, `ErrInvalidKey`. `ValidateKey`
  helper enforces the lowest-common-denominator key rules across
  providers.
- Adapters:
  - `s3`: aws-sdk-go-v2 backed; works with AWS S3, Cloudflare R2,
    MinIO, Backblaze B2, Tencent COS, and any S3-compatible store.
    Honours the standard AWS credential chain when keys are not
    supplied.
  - `aliyun`: Aliyun OSS Go SDK backed.
  - `memory`: in-process implementation for tests and quickstart code.
