// Package memory is an in-process implementation of [ossx.Bucket] for
// tests, dev sandboxes, and quickstart code paths. Not suitable for
// production: contents live in process memory only.
package memory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/foundation/ossx"
)

// New returns an empty in-memory bucket. Identifier is a label baked
// into presigned URLs so test assertions can tell two buckets apart;
// callers can leave it empty.
func New(identifier string) *Bucket {
	if identifier == "" {
		identifier = "memory"
	}
	return &Bucket{id: identifier, objects: make(map[string]object)}
}

// Bucket holds objects in a map. Safe for concurrent use.
type Bucket struct {
	id      string
	mu      sync.RWMutex
	objects map[string]object
}

type object struct {
	bytes        []byte
	contentType  string
	cacheControl string
	disposition  string
	metadata     map[string]string
	acl          ossx.ACL
	etag         string
	lastModified time.Time
}

// Put implements ossx.Bucket.
func (b *Bucket) Put(ctx context.Context, key string, r io.Reader, size int64, opts ossx.PutOptions) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	buf, err := readAll(r, size)
	if err != nil {
		return err
	}
	if opts.ACL == "" {
		opts.ACL = ossx.ACLPrivate
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.objects[key] = object{
		bytes:        buf,
		contentType:  opts.ContentType,
		cacheControl: opts.CacheControl,
		disposition:  opts.ContentDisposition,
		metadata:     copyMeta(opts.Metadata),
		acl:          opts.ACL,
		etag:         fmt.Sprintf(`"%x"`, len(buf)), // not real md5; tests don't need it
		lastModified: time.Now().UTC(),
	}
	return nil
}

// Get implements ossx.Bucket.
func (b *Bucket) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	o, ok := b.objects[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
	}
	return io.NopCloser(bytes.NewReader(o.bytes)), nil
}

// Stat implements ossx.Bucket.
func (b *Bucket) Stat(ctx context.Context, key string) (*ossx.ObjectInfo, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	o, ok := b.objects[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
	}
	return &ossx.ObjectInfo{
		Key:          key,
		Size:         int64(len(o.bytes)),
		ContentType:  o.contentType,
		ETag:         o.etag,
		LastModified: o.lastModified,
		Metadata:     copyMeta(o.metadata),
	}, nil
}

// Delete implements ossx.Bucket. Idempotent.
func (b *Bucket) Delete(ctx context.Context, key string) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.objects, key)
	return nil
}

// PresignGet returns a fake URL useful for round-trip assertions in
// tests. Real adapters return cryptographically signed URLs.
func (b *Bucket) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/memory: ttl must be > 0")
	}
	q := url.Values{
		"op":  []string{"get"},
		"ttl": []string{ttl.String()},
	}
	return fmt.Sprintf("memory://%s/%s?%s", b.id, key, q.Encode()), nil
}

// PresignPut returns a fake URL useful for round-trip assertions in
// tests.
func (b *Bucket) PresignPut(ctx context.Context, key string, ttl time.Duration, opts ossx.PresignPutOptions) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/memory: ttl must be > 0")
	}
	q := url.Values{
		"op":           []string{"put"},
		"ttl":          []string{ttl.String()},
		"content-type": []string{opts.ContentType},
	}
	return fmt.Sprintf("memory://%s/%s?%s", b.id, key, q.Encode()), nil
}

// readAll reads everything from r. If size >= 0 it preallocates.
func readAll(r io.Reader, size int64) ([]byte, error) {
	if size < 0 {
		return io.ReadAll(r)
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func copyMeta(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Compile-time check that Bucket satisfies the port.
var _ ossx.Bucket = (*Bucket)(nil)
