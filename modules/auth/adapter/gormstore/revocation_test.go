package gormstore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	"github.com/brizenchi/go-modules/modules/auth/domain"
)

func TestPersistentTokenRevocation(t *testing.T) {
	database := openTestDB(t)
	store := New(database)
	config := authjwt.Config{Secret: "revocation-test", Issuer: "test", Revocations: store}
	signer, err := authjwt.NewSigner(config)
	if err != nil {
		t.Fatal(err)
	}
	identity := domain.Identity{UserID: "user"}
	token, err := signer.Issue(identity, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	other, err := signer.Issue(identity, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if token.Value == other.Value {
		t.Fatal("independent tokens must have unique IDs")
	}
	if err := store.RevokeToken(t.Context(), strings.Repeat("a", 64), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := signer.Revoke(t.Context(), token.Value); err != nil {
			t.Fatal(err)
		}
	}
	var rows []tokenRevocationRow
	if err := database.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].TokenHash) != 64 || strings.Contains(rows[0].TokenHash, token.Value) {
		t.Fatalf("expected one hash-only row after cleanup, got %+v", rows)
	}
	config.Revocations = New(database)
	restarted, err := authjwt.NewSigner(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Parse(token.Value); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("revoked: %v", err)
	}
	if _, err := restarted.Parse(other.Value); err != nil {
		t.Fatalf("independent: %v", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := restarted.ParseContext(canceled, other.Value); !errors.Is(err, domain.ErrSessionUnavailable) || !errors.Is(err, context.Canceled) {
		t.Fatalf("verification lost request context: %v", err)
	}
	if err := restarted.Revoke(canceled, other.Value); !errors.Is(err, domain.ErrSessionUnavailable) {
		t.Fatalf("failed write reported success: %v", err)
	}
	if _, err := restarted.Parse(other.Value); err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&tokenRevocationRow{}).Where("token_hash = ?", rows[0].TokenHash).Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if revoked, err := store.IsRevoked(t.Context(), rows[0].TokenHash); err != nil || revoked {
		t.Fatalf("expired row: %v %v", revoked, err)
	}
}
