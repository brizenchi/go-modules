package saascore

import (
	"context"
	"testing"
)

func TestAuthStoreIncrDailyCount(t *testing.T) {
	db := newTestDB(t)
	if err := autoMigrateAuthStore(db); err != nil {
		t.Fatalf("autoMigrateAuthStore: %v", err)
	}

	store := newAuthStore(db)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		got, err := store.IncrDailyCount(ctx, "user@example.com", "2026-05-01")
		if err != nil {
			t.Fatalf("IncrDailyCount #%d: %v", i, err)
		}
		if got != i {
			t.Fatalf("IncrDailyCount #%d = %d, want %d", i, got, i)
		}
	}

	got, err := store.IncrDailyCount(ctx, "user@example.com", "2026-05-02")
	if err != nil {
		t.Fatalf("IncrDailyCount new bucket: %v", err)
	}
	if got != 1 {
		t.Fatalf("IncrDailyCount new bucket = %d, want 1", got)
	}
}
