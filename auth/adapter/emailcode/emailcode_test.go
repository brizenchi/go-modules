package emailcode

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/auth/adapter/memstore"
	"github.com/brizenchi/go-modules/auth/domain"
)

type captureMailer struct {
	mu    sync.Mutex
	calls int
	last  struct {
		ref  string
		to   []EmailAddress
		vars map[string]any
	}
}

func (m *captureMailer) SendProviderTemplate(ctx context.Context, ref string, to []EmailAddress, vars map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.last.ref = ref
	m.last.to = to
	m.last.vars = vars
	return nil
}

func TestIssuer_HappyPath(t *testing.T) {
	store := memstore.NewCodeStore()
	mailer := &captureMailer{}
	issuer := NewIssuer(Config{TemplateRef: "tpl-3"}, store, mailer)

	res, err := issuer.Issue(context.Background(), "User@Example.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if res.Email != "user@example.com" {
		t.Errorf("normalized email = %q", res.Email)
	}
	if mailer.calls != 1 {
		t.Errorf("mailer calls = %d, want 1", mailer.calls)
	}
	if mailer.last.ref != "tpl-3" {
		t.Errorf("template ref = %q", mailer.last.ref)
	}
	if _, ok := mailer.last.vars["code"].(string); !ok {
		t.Errorf("vars missing code: %v", mailer.last.vars)
	}
}

func TestIssuer_DebugReturnsCode(t *testing.T) {
	store := memstore.NewCodeStore()
	issuer := NewIssuer(Config{Debug: true}, store, nil)
	res, err := issuer.Issue(context.Background(), "a@b")
	if err != nil {
		t.Fatal(err)
	}
	if res.DebugCode == "" || len(res.DebugCode) != 6 {
		t.Errorf("debug code = %q", res.DebugCode)
	}
}

// failingMailer always returns an error from SendProviderTemplate.
type failingMailer struct{ err error }

func (m *failingMailer) SendProviderTemplate(ctx context.Context, ref string, to []EmailAddress, vars map[string]any) error {
	return m.err
}

// In debug mode, mailer failure must NOT fail Issue — the code is in the response.
// Without this behavior, restricted-IP or provider-down environments can't run E2E tests.
func TestIssuer_DebugTolerates_DeliveryFailure(t *testing.T) {
	store := memstore.NewCodeStore()
	mailer := &failingMailer{err: errors.New("brevo: 401 IP not whitelisted")}
	issuer := NewIssuer(Config{Debug: true, TemplateRef: "tpl-3"}, store, mailer)

	res, err := issuer.Issue(context.Background(), "a@b")
	if err != nil {
		t.Fatalf("debug mode should tolerate delivery failure, got %v", err)
	}
	if res.DebugCode == "" {
		t.Error("expected DebugCode set even when delivery fails in debug mode")
	}
}

// In production (debug=false), mailer failure must surface to the caller.
func TestIssuer_NonDebugFails_DeliveryFailure(t *testing.T) {
	store := memstore.NewCodeStore()
	mailer := &failingMailer{err: errors.New("smtp down")}
	issuer := NewIssuer(Config{Debug: false, TemplateRef: "tpl-3"}, store, mailer)

	_, err := issuer.Issue(context.Background(), "a@b")
	if err == nil {
		t.Fatal("expected delivery failure to surface in non-debug mode")
	}
}

func TestIssuer_RejectsInvalidEmail(t *testing.T) {
	issuer := NewIssuer(Config{}, memstore.NewCodeStore(), nil)
	_, err := issuer.Issue(context.Background(), "not-an-email")
	if !errors.Is(err, domain.ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestIssuer_PerMinuteRateLimit(t *testing.T) {
	issuer := NewIssuer(Config{Debug: true, MinResendGap: time.Hour}, memstore.NewCodeStore(), nil)
	if _, err := issuer.Issue(context.Background(), "a@b"); err != nil {
		t.Fatalf("first issue: %v", err)
	}
	_, err := issuer.Issue(context.Background(), "a@b")
	if !errors.Is(err, domain.ErrCodeRateLimited) {
		t.Errorf("expected rate-limited, got %v", err)
	}
}

func TestIssuer_DailyCap(t *testing.T) {
	issuer := NewIssuer(Config{Debug: true, MinResendGap: time.Nanosecond, DailyCap: 2}, memstore.NewCodeStore(), nil)
	for i := 0; i < 2; i++ {
		if _, err := issuer.Issue(context.Background(), "a@b"); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
		time.Sleep(2 * time.Nanosecond)
	}
	_, err := issuer.Issue(context.Background(), "a@b")
	if !errors.Is(err, domain.ErrCodeRateLimited) {
		t.Errorf("expected rate-limited after daily cap, got %v", err)
	}
}

func TestVerifier_HappyPath(t *testing.T) {
	store := memstore.NewCodeStore()
	issuer := NewIssuer(Config{Debug: true}, store, nil)
	verifier := NewVerifier(Config{}, store)
	res, _ := issuer.Issue(context.Background(), "a@b")
	if err := verifier.Verify(context.Background(), "a@b", res.DebugCode); err != nil {
		t.Errorf("verify: %v", err)
	}
	// Single-use: second verify must fail.
	if err := verifier.Verify(context.Background(), "a@b", res.DebugCode); !errors.Is(err, domain.ErrInvalidCode) {
		t.Errorf("expected single-use ErrInvalidCode, got %v", err)
	}
}

func TestVerifier_WrongCode(t *testing.T) {
	store := memstore.NewCodeStore()
	issuer := NewIssuer(Config{Debug: true}, store, nil)
	verifier := NewVerifier(Config{MaxAttempts: 3}, store)
	_, _ = issuer.Issue(context.Background(), "a@b")
	for i := 0; i < 2; i++ {
		if err := verifier.Verify(context.Background(), "a@b", "000000"); !errors.Is(err, domain.ErrInvalidCode) {
			t.Fatalf("attempt %d expected ErrInvalidCode, got %v", i, err)
		}
	}
	// Third attempt — hits cap, code should be invalidated.
	if err := verifier.Verify(context.Background(), "a@b", "000000"); !errors.Is(err, domain.ErrCodeMaxAttempts) {
		t.Errorf("expected ErrCodeMaxAttempts, got %v", err)
	}
}
