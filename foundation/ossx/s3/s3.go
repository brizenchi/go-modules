// Package s3 is an [ossx.Bucket] adapter backed by aws-sdk-go-v2.
//
// Compatible with AWS S3, Cloudflare R2, MinIO, Backblaze B2, Tencent
// COS, and any S3-compatible object store. Override [Config.Endpoint]
// to point at non-AWS backends; set [Config.UsePathStyle] for MinIO.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/brizenchi/go-modules/foundation/ossx"
)

// Config configures the adapter. AccessKeyID/SecretAccessKey are
// optional — if empty the default AWS credential chain is used
// (env, ~/.aws, IRSA, IMDS).
type Config struct {
	// Bucket name. Required.
	Bucket string
	// Region (e.g. "us-east-1", "auto" for R2). Required.
	Region string
	// Endpoint overrides the AWS endpoint for S3-compatible backends.
	// Examples:
	//   R2:    https://<account>.r2.cloudflarestorage.com
	//   MinIO: http://localhost:9000
	// Leave empty for AWS S3.
	Endpoint string
	// UsePathStyle forces path-style addressing
	// (https://endpoint/bucket/key vs https://bucket.endpoint/key).
	// MinIO and some on-prem stacks need this.
	UsePathStyle bool
	// AccessKeyID and SecretAccessKey, if both set, override the
	// default credential chain.
	AccessKeyID     string
	SecretAccessKey string
	// SessionToken — only set if using STS-issued temporary credentials.
	SessionToken string
}

// Bucket implements [ossx.Bucket] over an S3-compatible API.
type Bucket struct {
	cfg     Config
	client  *s3.Client
	presign *s3.PresignClient
}

// New constructs a Bucket from cfg.
func New(ctx context.Context, cfg Config) (*Bucket, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("ossx/s3: Bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("ossx/s3: Region is required")
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("ossx/s3: load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &Bucket{cfg: cfg, client: client, presign: s3.NewPresignClient(client)}, nil
}

// Put implements ossx.Bucket.
func (b *Bucket) Put(ctx context.Context, key string, r io.Reader, size int64, opts ossx.PutOptions) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	in := &s3.PutObjectInput{
		Bucket:        aws.String(b.cfg.Bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
	}
	if opts.ContentType != "" {
		in.ContentType = aws.String(opts.ContentType)
	}
	if opts.CacheControl != "" {
		in.CacheControl = aws.String(opts.CacheControl)
	}
	if opts.ContentDisposition != "" {
		in.ContentDisposition = aws.String(opts.ContentDisposition)
	}
	if len(opts.Metadata) > 0 {
		in.Metadata = opts.Metadata
	}
	switch opts.ACL {
	case ossx.ACLPublicRead:
		in.ACL = s3types.ObjectCannedACLPublicRead
	case ossx.ACLPrivate, "":
		in.ACL = s3types.ObjectCannedACLPrivate
	}
	_, err := b.client.PutObject(ctx, in)
	if err != nil {
		return fmt.Errorf("ossx/s3: put %q: %w", key, err)
	}
	return nil
}

// Get implements ossx.Bucket.
func (b *Bucket) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
		}
		return nil, fmt.Errorf("ossx/s3: get %q: %w", key, err)
	}
	return out.Body, nil
}

// Stat implements ossx.Bucket.
func (b *Bucket) Stat(ctx context.Context, key string) (*ossx.ObjectInfo, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return nil, err
	}
	out, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ossx.ErrNotFound, key)
		}
		return nil, fmt.Errorf("ossx/s3: stat %q: %w", key, err)
	}
	info := &ossx.ObjectInfo{Key: key, Metadata: out.Metadata}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	if out.ETag != nil {
		info.ETag = strings.Trim(*out.ETag, `"`)
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}
	return info, nil
}

// Delete implements ossx.Bucket. S3 delete is idempotent.
func (b *Bucket) Delete(ctx context.Context, key string) error {
	if err := ossx.ValidateKey(key); err != nil {
		return err
	}
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("ossx/s3: delete %q: %w", key, err)
	}
	return nil
}

// PresignGet implements ossx.Bucket.
func (b *Bucket) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/s3: ttl must be > 0")
	}
	out, err := b.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("ossx/s3: presign get %q: %w", key, err)
	}
	return out.URL, nil
}

// PresignPut implements ossx.Bucket.
func (b *Bucket) PresignPut(ctx context.Context, key string, ttl time.Duration, opts ossx.PresignPutOptions) (string, error) {
	if err := ossx.ValidateKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ossx/s3: ttl must be > 0")
	}
	in := &s3.PutObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	}
	if opts.ContentType != "" {
		in.ContentType = aws.String(opts.ContentType)
	}
	out, err := b.presign.PresignPutObject(ctx, in, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("ossx/s3: presign put %q: %w", key, err)
	}
	return out.URL, nil
}

// isNotFound recognises S3's missing-key errors. The SDK uses two
// shapes depending on the call (NoSuchKey for GET, NotFound for HEAD).
func isNotFound(err error) bool {
	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *s3types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	return false
}

// Compile-time check that Bucket satisfies the port.
var _ ossx.Bucket = (*Bucket)(nil)
