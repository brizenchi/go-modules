package rdx_test

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/foundation/rdx"
)

// Example_acquire shows the "make sure exactly one worker does X for
// the next N seconds" pattern. Returns immediately if another caller
// already holds the lock. Always Unlock — the Lua-scripted release
// won't accidentally drop a lock taken over by someone else after
// TTL expiry.
func Example_acquire() {
	ctx := context.Background()
	client, _ := rdx.Open(ctx, rdx.Config{Addr: "localhost:6379"})

	lock, ok, err := rdx.Acquire(ctx, client, "send-daily-report", 30*time.Second)
	if err != nil {
		return
	}
	if !ok {
		return // another worker has it
	}
	defer lock.Unlock(ctx)

	// ... do the work ...
}
