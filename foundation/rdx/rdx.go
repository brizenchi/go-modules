// Package rdx is a thin Redis client helper standardized across services.
//
// Provides:
//   - Open(cfg) → *redis.Client (the upstream go-redis/v9 client)
//   - HealthCheck(ctx, client)
//   - A small Lock primitive for "do this work at most once" workflows
//
// Stdlib + go-redis/v9 only.
package rdx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config configures the client.
type Config struct {
	// Addr: "host:port". Required.
	Addr     string
	Password string
	DB       int

	// Pool tuning. Sane defaults applied if zero.
	PoolSize       int           // default 10
	MinIdleConns   int           // default 2
	DialTimeout    time.Duration // default 5s
	ReadTimeout    time.Duration // default 3s
	WriteTimeout   time.Duration // default 3s

	// KeyPrefix is prepended to every key by Lock helpers and by the
	// caller via Prefix(). Empty disables prefixing. Useful when one
	// Redis serves many environments/services.
	KeyPrefix string
}

// Open builds the client and verifies connectivity once before returning.
func Open(ctx context.Context, cfg Config) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("rdx: addr required")
	}
	cli := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     nonZeroInt(cfg.PoolSize, 10),
		MinIdleConns: nonZeroInt(cfg.MinIdleConns, 2),
		DialTimeout:  nonZeroDur(cfg.DialTimeout, 5*time.Second),
		ReadTimeout:  nonZeroDur(cfg.ReadTimeout, 3*time.Second),
		WriteTimeout: nonZeroDur(cfg.WriteTimeout, 3*time.Second),
	})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("rdx: ping: %w", err)
	}
	return cli, nil
}

// HealthCheck pings the server with a short timeout.
func HealthCheck(ctx context.Context, cli *redis.Client) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return cli.Ping(pingCtx).Err()
}

// Prefix returns key with cfg's KeyPrefix prepended (or key unchanged
// when prefix is empty). Use for namespacing keys per service.
func Prefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + ":" + key
}

func nonZeroInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
func nonZeroDur(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}

// avoid unused import noise when test builds skip features
var _ = errors.Is
