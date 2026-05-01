// Package ossx is the object-storage abstraction shared across services.
//
// The interface deliberately stays small — just the operations every
// SaaS reaches for: upload, download, stat, delete, and the two presign
// flavours that let browsers upload/download directly without
// proxying bytes through your API.
//
// Callers depend on [Bucket] and never on a specific provider SDK.
// Adapters live in subpackages:
//
//   - foundation/ossx/s3      — AWS S3, Cloudflare R2, MinIO, any S3-compatible
//   - foundation/ossx/aliyun  — Aliyun OSS
//   - foundation/ossx/memory  — in-process, useful for tests
package ossx

import (
	"context"
	"errors"
	"io"
	"time"
)

// Bucket is the operations every adapter implements.
//
// Implementations must be safe for concurrent use by multiple
// goroutines. Errors that mean "no such object" must wrap [ErrNotFound]
// so callers can use errors.Is to discriminate.
type Bucket interface {
	// Put uploads the bytes from r as object key. size must match the
	// number of bytes available from r — most providers require it
	// up-front so they can compute Content-Length. Pass -1 only if the
	// adapter explicitly documents support for it.
	Put(ctx context.Context, key string, r io.Reader, size int64, opts PutOptions) error

	// Get returns a reader for the object's bytes. The caller MUST close
	// the returned ReadCloser. Returns ErrNotFound when the key is
	// absent.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Stat returns metadata for an object without downloading it.
	// Returns ErrNotFound when the key is absent.
	Stat(ctx context.Context, key string) (*ObjectInfo, error)

	// Delete removes an object. Returns nil even if the key didn't
	// exist (idempotent), to match S3 / Aliyun semantics.
	Delete(ctx context.Context, key string) error

	// PresignGet returns a time-limited URL that downloads the object.
	// Pass it to a browser / client to bypass your API for reads.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)

	// PresignPut returns a time-limited URL the client can PUT to in
	// order to upload directly. opts.ContentType is signed into the
	// URL — the client must send the same Content-Type header.
	PresignPut(ctx context.Context, key string, ttl time.Duration, opts PresignPutOptions) (string, error)
}

// PutOptions are upload-time hints. Adapters apply what they can and
// silently ignore what they can't.
type PutOptions struct {
	// ContentType set on the stored object (e.g. "image/png"). If empty,
	// adapters fall back to provider defaults (typically
	// "application/octet-stream").
	ContentType string
	// CacheControl sets the Cache-Control header served on GET.
	// e.g. "public, max-age=31536000, immutable".
	CacheControl string
	// ContentDisposition controls download filename.
	// e.g. `attachment; filename="report.pdf"`.
	ContentDisposition string
	// Metadata is arbitrary user metadata. Provider-prefixed (e.g.
	// x-amz-meta-) automatically. Keys must be ASCII; values must be
	// printable.
	Metadata map[string]string
	// ACL controls object visibility. Empty defaults to ACLPrivate.
	ACL ACL
}

// PresignPutOptions are signed into a presigned upload URL. The client
// must send a request that matches these exactly, otherwise the
// provider rejects the upload.
type PresignPutOptions struct {
	ContentType string
}

// ObjectInfo is metadata returned by Stat.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

// ACL controls object visibility.
type ACL string

const (
	// ACLPrivate makes the object readable only via authenticated APIs
	// or presigned URLs. Default.
	ACLPrivate ACL = "private"
	// ACLPublicRead makes the object readable by anyone who knows the
	// URL. Use sparingly — once an object is public, scrapers will
	// find it.
	ACLPublicRead ACL = "public-read"
)

// Sentinel errors. Adapters wrap these so callers can use errors.Is.
var (
	// ErrNotFound — the requested key does not exist.
	ErrNotFound = errors.New("ossx: object not found")
	// ErrInvalidKey — the key is malformed (empty, leading slash, etc.).
	ErrInvalidKey = errors.New("ossx: invalid key")
)

// ValidateKey enforces the lowest common denominator across providers:
// non-empty, no leading slash, ≤ 1024 bytes (S3 limit). Adapters call
// this so behaviour stays consistent.
func ValidateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if key[0] == '/' {
		return ErrInvalidKey
	}
	if len(key) > 1024 {
		return ErrInvalidKey
	}
	return nil
}
