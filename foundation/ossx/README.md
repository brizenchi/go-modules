# foundation/ossx

> Object storage abstraction shared across services. One `Bucket` interface, three adapters.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/ossx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/ossx)

Wraps the operations every SaaS reaches for: upload, download, stat,
delete, and presigned GET/PUT URLs (so browsers upload/download
directly without proxying bytes through your API).

Adapters live in subpackages. Pick the one that matches your provider:

| Subpackage                                        | Provider                                                       |
|---------------------------------------------------|----------------------------------------------------------------|
| [`foundation/ossx/s3`](./s3/)                     | AWS S3, Cloudflare R2, MinIO, Backblaze B2, Tencent COS, any S3-compatible |
| [`foundation/ossx/aliyun`](./aliyun/)             | Aliyun OSS                                                     |
| [`foundation/ossx/memory`](./memory/)             | In-process — for tests and dev sandboxes                       |

Callers depend only on `ossx.Bucket`; the adapter is constructed at
boot and injected.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/ossx
```

## Quick start

```go
import (
    "context"
    "strings"
    "time"

    "github.com/brizenchi/go-modules/foundation/ossx"
    s3oss "github.com/brizenchi/go-modules/foundation/ossx/s3"
)

ctx := context.Background()

bucket, err := s3oss.New(ctx, s3oss.Config{
    Bucket: "my-app",
    Region: "us-east-1",
    // For Cloudflare R2:
    //   Region:   "auto",
    //   Endpoint: "https://<account>.r2.cloudflarestorage.com",
    // For MinIO:
    //   Endpoint: "http://localhost:9000", UsePathStyle: true,
})

// Upload
_ = bucket.Put(ctx, "users/42/avatar.png",
    strings.NewReader(pngBytes), int64(len(pngBytes)),
    ossx.PutOptions{
        ContentType: "image/png",
        CacheControl: "public, max-age=31536000",
        ACL: ossx.ACLPublicRead,
    })

// Download
rc, err := bucket.Get(ctx, "users/42/avatar.png")
defer rc.Close()

// Stat
info, err := bucket.Stat(ctx, "users/42/avatar.png")
fmt.Println(info.Size, info.ETag)

// Presign — let the browser GET directly
url, _ := bucket.PresignGet(ctx, "reports/q1.pdf", 5*time.Minute)

// Presign — let the browser PUT directly (skip your API for big uploads)
upload, _ := bucket.PresignPut(ctx, "uploads/raw.zip", 15*time.Minute,
    ossx.PresignPutOptions{ContentType: "application/zip"})
```

## API

```go
type Bucket interface {
    Put(ctx, key, r, size, opts) error
    Get(ctx, key) (io.ReadCloser, error)
    Stat(ctx, key) (*ObjectInfo, error)
    Delete(ctx, key) error                            // idempotent
    PresignGet(ctx, key, ttl) (string, error)
    PresignPut(ctx, key, ttl, opts) (string, error)
}
```

| Type                   | Purpose                                                       |
|------------------------|---------------------------------------------------------------|
| `PutOptions`           | ContentType, CacheControl, ContentDisposition, Metadata, ACL  |
| `PresignPutOptions`    | ContentType (signed into the URL)                             |
| `ObjectInfo`           | Key, Size, ContentType, ETag, LastModified, Metadata          |
| `ACL`                  | `ACLPrivate` (default), `ACLPublicRead`                       |

Sentinel errors (`errors.Is`-friendly):

| Error           | When                                  |
|-----------------|---------------------------------------|
| `ErrNotFound`   | Get/Stat against a missing key        |
| `ErrInvalidKey` | Empty, leading-slash, or > 1024 bytes |

## Provider notes

### S3 adapter

Credentials follow the standard AWS chain (env, `~/.aws`, IRSA, IMDS)
unless you set `Config.AccessKeyID` + `SecretAccessKey` explicitly.

| Backend         | `Region`           | `Endpoint`                                        | `UsePathStyle` |
|-----------------|--------------------|---------------------------------------------------|----------------|
| AWS S3          | `us-east-1` etc.   | (empty)                                           | false          |
| Cloudflare R2   | `auto`             | `https://<account>.r2.cloudflarestorage.com`     | false          |
| MinIO           | any                | `http://localhost:9000`                          | true           |
| Backblaze B2    | `us-west-002`      | `https://s3.us-west-002.backblazeb2.com`         | false          |
| Tencent COS     | `ap-guangzhou`     | `https://cos.<region>.myqcloud.com`              | false          |

### Aliyun adapter

```go
import aliossx "github.com/brizenchi/go-modules/foundation/ossx/aliyun"

bucket, err := aliossx.New(aliossx.Config{
    Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
    Bucket:          "my-app",
    AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
    AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
})
```

### Memory adapter

Stores objects in a `map[string][]byte`; data is lost when the process
exits. Useful for tests and quickstart code.

```go
import memoss "github.com/brizenchi/go-modules/foundation/ossx/memory"

bucket := memoss.New("test")
```

The presigned URLs returned by the memory adapter are fake
(`memory://...`); they round-trip query-string parameters so you can
write assertions, but a real client cannot upload through them.

## Sizing & key conventions

- Keys are validated at the port: non-empty, no leading slash, ≤ 1024
  bytes (S3 limit). Adapters apply the same rule for portability.
- Always pass an accurate `size` to `Put`. Most providers require
  Content-Length up-front and will reject `-1` unless you read the
  adapter's docs (the memory adapter accepts `-1`).
- For uploads larger than ~100MB, prefer `PresignPut` and let the
  client upload directly — your server doesn't need to relay bytes.

## Testing strategy

The port and memory adapter are unit-tested. The S3 and Aliyun
adapters are exercised through compile-time interface assertions
(`var _ ossx.Bucket = (*Bucket)(nil)`) but not against live providers
in this repo's CI — integration testing happens in consuming projects
where credentials are available.

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
