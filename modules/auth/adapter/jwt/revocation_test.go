package jwt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type failingRevocations struct{ reads, writes int }

func (store *failingRevocations) IsRevoked(context.Context, string) (bool, error) {
	store.reads++
	return false, errors.New("offline")
}

func (store *failingRevocations) RevokeToken(context.Context, string, time.Time) error {
	store.writes++
	return errors.New("offline")
}

func TestRevocationValidatesBeforeAccessingStore(t *testing.T) {
	store := &failingRevocations{}
	signer, err := NewSigner(Config{Secret: "revocation-test", Issuer: "expected", Revocations: store})
	if err != nil {
		t.Fatal(err)
	}
	for _, config := range []Config{
		{Secret: "other-secret", Issuer: "expected"},
		{Secret: "revocation-test", Issuer: "other-issuer"},
		{Secret: "revocation-test", Issuer: "expected"},
	} {
		otherSigner, err := NewSigner(config)
		if err != nil {
			t.Fatal(err)
		}
		identity := domain.Identity{UserID: "user"}
		if config.Secret == "revocation-test" && config.Issuer == "expected" {
			identity.UserID = ""
		}
		token, err := otherSigner.Issue(identity, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := signer.Parse(token.Value); !errors.Is(err, domain.ErrInvalidToken) {
			t.Fatalf("invalid token: %v", err)
		}
		if err := signer.Revoke(t.Context(), token.Value); !errors.Is(err, domain.ErrInvalidToken) {
			t.Fatalf("invalid revoke: %v", err)
		}
	}
	if store.reads != 0 || store.writes != 0 {
		t.Fatal("invalid credentials reached revocation store")
	}
	token, err := signer.Issue(domain.Identity{UserID: "user"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Parse(token.Value); !errors.Is(err, domain.ErrSessionUnavailable) {
		t.Fatalf("read failure: %v", err)
	}
	if err := signer.Revoke(t.Context(), token.Value); !errors.Is(err, domain.ErrSessionUnavailable) {
		t.Fatalf("write failure: %v", err)
	}
	withoutStore, err := NewSigner(Config{Secret: "revocation-test", Issuer: "expected"})
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutStore.Revoke(t.Context(), token.Value); !errors.Is(err, domain.ErrSessionUnavailable) {
		t.Fatalf("unconfigured store: %v", err)
	}
}

func TestRevocationRejectsAlternateEncodingAndExpiredOrTicketTokens(t *testing.T) {
	store := &failingRevocations{}
	signer, err := NewSigner(Config{Secret: "canonical-token-test", Revocations: store})
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.Issue(domain.Identity{UserID: "user"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, token.Value[len(token.Value)-1])
	alternate := token.Value[:len(token.Value)-1] + string(alphabet[last^1])
	invalidTokens := []string{alternate, token.Value + "\n"}
	for _, claims := range []userClaims{
		{RegisteredClaims: jwtv5.RegisteredClaims{Subject: "user", ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(-time.Hour))}, Type: tokenTypeAccess},
		{RegisteredClaims: jwtv5.RegisteredClaims{Subject: "user", ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}, Type: tokenTypeWSTicket},
	} {
		value, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte("canonical-token-test"))
		if err != nil {
			t.Fatal(err)
		}
		invalidTokens = append(invalidTokens, value)
	}
	for _, value := range invalidTokens {
		if _, err := signer.Parse(value); !errors.Is(err, domain.ErrInvalidToken) {
			t.Fatalf("alternate or invalid token accepted: %v", err)
		}
		if err := signer.Revoke(t.Context(), value); !errors.Is(err, domain.ErrInvalidToken) {
			t.Fatalf("invalid revoke: %v", err)
		}
	}
	if store.reads != 0 || store.writes != 0 {
		t.Fatal("invalid credentials reached store")
	}
}
