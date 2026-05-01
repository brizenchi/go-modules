// Package aliyun is an [ossx.Bucket] adapter backed by the Aliyun OSS
// Go SDK. Use this when serving traffic from mainland China; for the
// rest of the world prefer the s3 adapter.
package aliyun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	aliyun "github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/brizenchi/go-modules/foundation/ossx"
)

// Config configures the adapter.
type Config struct {
	// Endpoint is the OSS endpoint, e.g. "oss-cn-hangzhou.aliyuncs.com".
	// Required.
	Endpoint string
	// Bucket name. Required.
	Bucket string
	// AccessKeyID and AccessKeySecret. Required.
	AccessKeyID     string
	AccessKeySecret string
	// SecurityToken — only set if using STS-issued temporary credentials.
	SecurityToken string
}

// Bucket implements [ossx.Bucket] backed by Aliyun OSS.
type Bucket struct {
	cfg    Config
	bucket *aliyun.Bucket
}

// New constructs a Bucket from cfg.
func New(cfg Config) (*Bucket, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("ossx/aliyun: Endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("ossx/aliyun: Bucket is required")
	}
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("ossx/aliyun: AccessKeyID and AccessKeySecret are required")
	}
	clientOpts := []aliyun.ClientOption{}
	if cfg.SecurityToken != "" {
		clientOpts = append(clientOpts, aliyun.SecurityToken(cfg.SecurityToken))
	}
	client, err := aliyun.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("ossx/aliyun: client: %w", err)
	}
	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("ossx/aliyun: bucket: %w", err)
	}
	return &Bucket{cfg: cfg, bucket: bucket}, nil
}

// Put implements ossx.Bucket. ctx is honoured via the SDK's context option.
func (b *Bucket) Put(ctx context.Context, key string, r io.Reader, size int64, opts ossx.PutOptions) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	putOpts := []aliyun.Option{aliyun.WithContext(ctx)}
	if size >= 0 {
		putOpts = append(putOpts, aliyun.ContentLength(size))
	}
	if opts.ContentType != "" {
		putOpts = append(putOpts, aliyun.ContentType(opts.ContentType))
	}
	if opts.CacheControl != "" {
		putOpts = append(putOpts, aliyun.CacheControl(opts.CacheControl))
	}
	if opts.ContentDisposition != "" {
		putOpts = append(putOpts, aliyun.ContentDisposition(opts.ContentDisposition))
	}
	for k, v := range opts.Metadata {
		putOpts = append(putOpts, aliyun.Meta(k, v))
	}
	if acl := mapACL(opts.ACL); acl != "" {
		putOpts = append(putOpts, aliyun.ObjectACL(acl))
	}
	if err := b.bucket.PutObject(key, r, putOpts...); err != nil {
		return fmt.Errorf("ossx/aliyun: put %q: %w", key, err)
	}
	return nil
}

// Get implements ossx.Bucket.
func (b *Bucket) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	rc, err := b.bucket.GetObject(key, aliyun.WithContext(ctx))
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
		}
		return nil, fmt.Errorf("ossx/aliyun: get %q: %w", key, err)
	}
	return rc, nil
}

// Stat implements ossx.Bucket.
func (b *Bucket) Stat(ctx context.Context, key string) (*ossx.ObjectInfo, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	hdr, err := b.bucket.GetObjectMeta(key, aliyun.WithContext(ctx))
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
		}
		return nil, fmt.Errorf("ossx/aliyun: stat %q: %w", key, err)
	}
	info := &ossx.ObjectInfo{Key: key}
	if v := hdr.Get(aliyun.HTTPHeaderContentLength); v != "" {
		fmt.Sscanf(v, "%d", &info.Size)
	}
	info.ContentType = hdr.Get(aliyun.HTTPHeaderContentType)
	info.ETag = strings.Trim(hdr.Get(aliyun.HTTPHeaderEtag), `"`)
	if v := hdr.Get(aliyun.HTTPHeaderLastModified); v != "" {
		if t, err := time.Parse(time.RFC1123, v); err == nil {
			info.LastModified = t
		}
	}
	const metaPrefix = "X-Oss-Meta-"
	for k, vs := range hdr {
		if strings.HasPrefix(k, metaPrefix) && len(vs) > 0 {
			if info.Metadata == nil {
				info.Metadata = make(map[string]string)
			}
			info.Metadata[strings.ToLower(k[len(metaPrefix):])] = vs[0]
		}
	}
	return info, nil
}

// Delete implements ossx.Bucket. Aliyun OSS DeleteObject is idempotent.
func (b *Bucket) Delete(ctx context.Context, key string) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	if err := b.bucket.DeleteObject(key, aliyun.WithContext(ctx)); err != nil {
		return fmt.Errorf("ossx/aliyun: delete %q: %w", key, err)
	}
	return nil
}

// PresignGet implements ossx.Bucket.
func (b *Bucket) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/aliyun: ttl must be > 0")
	}
	url, err := b.bucket.SignURL(key, aliyun.HTTPGet, int64(ttl/time.Second))
	if err != nil {
		return "", fmt.Errorf("ossx/aliyun: presign get %q: %w", key, err)
	}
	return url, nil
}

// PresignPut implements ossx.Bucket.
func (b *Bucket) PresignPut(ctx context.Context, key string, ttl time.Duration, opts ossx.PresignPutOptions) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/aliyun: ttl must be > 0")
	}
	signOpts := []aliyun.Option{}
	if opts.ContentType != "" {
		signOpts = append(signOpts, aliyun.ContentType(opts.ContentType))
	}
	url, err := b.bucket.SignURL(key, aliyun.HTTPPut, int64(ttl/time.Second), signOpts...)
	if err != nil {
		return "", fmt.Errorf("ossx/aliyun: presign put %q: %w", key, err)
	}
	return url, nil
}

func mapACL(in ossx.ACL) aliyun.ACLType {
	switch in {
	case ossx.ACLPublicRead:
		return aliyun.ACLPublicRead
	case ossx.ACLPrivate:
		return aliyun.ACLPrivate
	}
	return ""
}

// isNotFound recognises Aliyun's NoSuchKey error.
func isNotFound(err error) bool {
	var se aliyun.ServiceError
	if errors.As(err, &se) {
		return se.StatusCode == 404 || se.Code == "NoSuchKey"
	}
	return false
}

// Compile-time check that Bucket satisfies the port.
var _ ossx.Bucket = (*Bucket)(nil)
