package note

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNoteValidationAndOwnership(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&Note{}); err != nil {
		t.Fatal(err)
	}
	svc := newService(newRepository(db))
	for _, input := range []struct{ title, body string }{
		{strings.Repeat("字", 201), ""}, {"valid", strings.Repeat("文", 50001)}, {string([]byte{0xff}), ""},
	} {
		if _, err := svc.Create(t.Context(), "alice", input.title, input.body); !errors.Is(err, ErrInvalidContent) {
			t.Fatalf("expected content validation error, got %v", err)
		}
	}
	if _, err := svc.Create(t.Context(), "alice", "  ", "body"); !errors.Is(err, ErrEmptyTitle) {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), "alice", "  我的笔记  ", strings.Repeat("文", 50000)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), "bob", "Private", "secret"); err != nil {
		t.Fatal(err)
	}
	rows, err := svc.ListMine(t.Context(), "alice")
	if err != nil || len(rows) != 1 || rows[0].Title != "我的笔记" {
		t.Fatalf("unexpected user notes: %+v, %v", rows, err)
	}
}
