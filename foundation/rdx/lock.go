package rdx

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLockNotHeld is returned by Unlock when the lock was already gone
// (TTL expired or someone else released).
var ErrLockNotHeld = errors.New("rdx: lock not held")

// Lock is a simple distributed lock backed by SET NX PX.
//
// NOT a re-entrant lock. NOT designed for high-contention critical
// sections — for those, use Redlock or a real coordinator. This is
// "make sure exactly one worker does X for the next N seconds" territory.
type Lock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// Acquire attempts to take the lock for ttl. Returns (lock, true) on
// success, (nil, false) when already held by another caller.
func Acquire(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (*Lock, bool, error) {
	if ttl <= 0 {
		return nil, false, errors.New("rdx: ttl must be > 0")
	}
	value, err := randomToken()
	if err != nil {
		return nil, false, err
	}
	// SetNX with TTL maps to SET NX PX in go-redis/v9. The SA1019
	// notice on staticcheck refers to the Redis protocol command
	// "SETNX" (which has no TTL), not this go-redis helper.
	ok, err := client.SetNX(ctx, key, value, ttl).Result() //nolint:staticcheck // SA1019: see comment above
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &Lock{client: client, key: key, value: value, ttl: ttl}, true, nil
}

// Unlock releases the lock — but only if we still hold it. Uses a Lua
// script to make compare-and-delete atomic; otherwise a slow caller
// might delete a lock that has already been re-acquired by someone else.
func (l *Lock) Unlock(ctx context.Context) error {
	const script = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`
	res, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Int()
	if err != nil {
		return err
	}
	if res == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// Refresh extends the lock's TTL — only if still held.
func (l *Lock) Refresh(ctx context.Context, ttl time.Duration) error {
	const script = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("PEXPIRE", KEYS[1], ARGV[2]) else return 0 end`
	res, err := l.client.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if res == 0 {
		return ErrLockNotHeld
	}
	l.ttl = ttl
	return nil
}

// Key returns the locked key (for logging / debugging).
func (l *Lock) Key() string { return l.key }

// randomToken returns a 32-char unguessable token (lock fencing value).
func randomToken() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const n = 32
	buf := make([]byte, n)
	if _, err := cryptoReader.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}
