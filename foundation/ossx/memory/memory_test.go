package memory_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/foundation/ossx"
	"github.com/brizenchi/go-modules/foundation/ossx/memory"
)

func TestPutGetRoundtrip(t *testing.T) {
	ctx := context.Background()
	b := memory.New("test")

	body := []byte("hello, ossx")
	if err := b.Put(ctx, "greeting.txt", strings.NewReader(string(body)), int64(len(body)), ossx.PutOptions{
		ContentType: "text/plain",
		Metadata:    map[string]string{"author": "alice"},
	}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := b.Get(ctx, "greeting.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}

	info, err := b.Stat(ctx, "greeting.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != int64(len(body)) {
		t.Fatalf("Stat size = %d, want %d", info.Size, len(body))
	}
	if info.ContentType != "text/plain" {
		t.Fatalf("Stat content-type = %q", info.ContentType)
	}
	if info.Metadata["author"] != "alice" {
		t.Fatalf("Stat metadata lost: %+v", info.Metadata)
	}
}

func TestGetMissingReturnsErrNotFound(t *testing.T) {
	b := memory.New("")
	_, err := b.Get(context.Background(), "missing.txt")
	if !errors.Is(err, ossx.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	_, err = b.Stat(context.Background(), "missing.txt")
	if !errors.Is(err, ossx.ErrNotFound) {
		t.Fatalf("Stat got %v, want ErrNotFound", err)
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	ctx := context.Background()
	b := memory.New("")
	_ = b.Put(ctx, "tmp", strings.NewReader("x"), 1, ossx.PutOptions{})
	if err := b.Delete(ctx, "tmp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := b.Delete(ctx, "tmp"); err != nil {
		t.Fatalf("second Delete returned %v, want nil", err)
	}
}

func TestPresignURLsCarryMetadata(t *testing.T) {
	b := memory.New("acme")
	ctx := context.Background()

	get, err := b.PresignGet(ctx, "k", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if !strings.Contains(get, "memory://acme/k") || !strings.Contains(get, "op=get") {
		t.Fatalf("PresignGet url unexpected: %s", get)
	}

	put, err := b.PresignPut(ctx, "k", time.Minute, ossx.PresignPutOptions{ContentType: "image/png"})
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if !strings.Contains(put, "op=put") || !strings.Contains(put, "image%2Fpng") {
		t.Fatalf("PresignPut url unexpected: %s", put)
	}

	if _, err := b.PresignGet(ctx, "k", 0); err == nil {
		t.Fatal("expected error for ttl=0")
	}
}

func TestInvalidKey(t *testing.T) {
	ctx := context.Background()
	b := memory.New("")
	if err := b.Put(ctx, "", strings.NewReader(""), 0, ossx.PutOptions{}); !errors.Is(err, ossx.ErrInvalidKey) {
		t.Fatalf("got %v, want ErrInvalidKey", err)
	}
	if err := b.Put(ctx, "/leading", strings.NewReader(""), 0, ossx.PutOptions{}); !errors.Is(err, ossx.ErrInvalidKey) {
		t.Fatalf("got %v, want ErrInvalidKey", err)
	}
}
