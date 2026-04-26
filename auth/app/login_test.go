package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/event"
)

func TestLogin_VerifyCode_NewUserPublishesSignedUpAndLoggedIn(t *testing.T) {
	users := newMockUserStore()
	bus := &mockBus{}
	svc := NewLoginService(LoginDeps{
		Issuer:   &mockIssuer{},
		Verifier: &mockVerifier{},
		Users:    users,
		Signer:   mockSigner{},
		Bus:      bus,
	})

	res, err := svc.VerifyCode(context.Background(), "new@example.com", "good")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.Token.Value != "tok-u-new@example.com" {
		t.Errorf("token = %q", res.Token.Value)
	}
	if !res.Identity.IsNew {
		t.Error("expected IsNew=true on first login")
	}
	if got := len(bus.Got(event.KindUserSignedUp)); got != 1 {
		t.Errorf("UserSignedUp count = %d, want 1", got)
	}
	if got := len(bus.Got(event.KindUserLoggedIn)); got != 1 {
		t.Errorf("UserLoggedIn count = %d, want 1", got)
	}
	if users.logins["u-new@example.com"] != 1 {
		t.Errorf("MarkLogin not called")
	}
}

func TestLogin_VerifyCode_ReturningUserNoSignedUp(t *testing.T) {
	users := newMockUserStore()
	users.seed(domain.Identity{UserID: "u1", Email: "old@example.com"})
	bus := &mockBus{}
	svc := NewLoginService(LoginDeps{
		Issuer:   &mockIssuer{},
		Verifier: &mockVerifier{},
		Users:    users,
		Signer:   mockSigner{},
		Bus:      bus,
	})

	res, err := svc.VerifyCode(context.Background(), "old@example.com", "good")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.Identity.IsNew {
		t.Error("expected IsNew=false on returning user")
	}
	if got := len(bus.Got(event.KindUserSignedUp)); got != 0 {
		t.Errorf("UserSignedUp = %d, want 0 for returning user", got)
	}
	if got := len(bus.Got(event.KindUserLoggedIn)); got != 1 {
		t.Errorf("UserLoggedIn = %d, want 1", got)
	}
}

func TestLogin_VerifyCode_BadCode(t *testing.T) {
	bus := &mockBus{}
	svc := NewLoginService(LoginDeps{
		Issuer:   &mockIssuer{},
		Verifier: &mockVerifier{},
		Users:    newMockUserStore(),
		Signer:   mockSigner{},
		Bus:      bus,
	})
	_, err := svc.VerifyCode(context.Background(), "u@b", "wrong")
	if !errors.Is(err, domain.ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode, got %v", err)
	}
	if len(bus.Got(event.KindUserLoggedIn)) != 0 {
		t.Error("no event should publish on failed verify")
	}
}

func TestLogin_SendCode_DelegatesToIssuer(t *testing.T) {
	issuer := &mockIssuer{}
	svc := NewLoginService(LoginDeps{Issuer: issuer})
	if _, err := svc.SendCode(context.Background(), "a@b"); err != nil {
		t.Fatal(err)
	}
	if issuer.calls != 1 {
		t.Errorf("issuer calls = %d", issuer.calls)
	}
}

func TestLogin_SendCode_NoIssuer(t *testing.T) {
	svc := NewLoginService(LoginDeps{})
	_, err := svc.SendCode(context.Background(), "a@b")
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}
