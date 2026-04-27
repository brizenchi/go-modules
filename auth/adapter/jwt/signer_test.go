package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
)

func newTestSigner(t *testing.T) *Signer {
	t.Helper()
	s, err := NewSigner(Config{Secret: "test-secret-0123456789", Issuer: "test", UserTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSigner_RoundTrip(t *testing.T) {
	s := newTestSigner(t)
	id := domain.Identity{
		UserID: "user-42",
		Email:  "u@example.com",
		Role:   domain.RoleAdmin,
	}
	tok, err := s.Issue(id, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tok.Value == "" || tok.ExpiresAt.Before(time.Now()) {
		t.Fatalf("invalid token: %+v", tok)
	}
	parsed, err := s.Parse(tok.Value)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.UserID != "user-42" || parsed.Email != "u@example.com" || parsed.Role != domain.RoleAdmin {
		t.Errorf("parsed = %+v", parsed)
	}
}

func TestSigner_RejectsTamperedToken(t *testing.T) {
	s := newTestSigner(t)
	tok, _ := s.Issue(domain.Identity{UserID: "u1", Role: domain.RoleUser}, time.Hour)
	// Mutate a char in the payload segment of the JWT (between the two
	// dots). Mutating the last sig char is unreliable: base64url's
	// trailing char encodes only 4 bits, so swapping to a value sharing
	// the high 2 bits leaves the decoded signature unchanged → HMAC
	// verify still passes.
	parts := splitDots(tok.Value)
	if len(parts) != 3 {
		t.Fatalf("malformed jwt: %q", tok.Value)
	}
	mid := []byte(parts[1])
	idx := len(mid) / 2
	if mid[idx] == 'A' {
		mid[idx] = 'B'
	} else {
		mid[idx] = 'A'
	}
	bad := parts[0] + "." + string(mid) + "." + parts[2]
	_, err := s.Parse(bad)
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestSigner_RejectsExpiredToken(t *testing.T) {
	// JWT exp is second-precision; sleep past a full second to avoid
	// flake when nanosecond TTLs round to the same second as now.
	s := newTestSigner(t)
	tok, err := s.Issue(domain.Identity{UserID: "u1"}, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	_, err = s.Parse(tok.Value)
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken (expired), got %v", err)
	}
}

func TestSigner_RejectsWrongSecret(t *testing.T) {
	s1, _ := NewSigner(Config{Secret: "secret-1", UserTTL: time.Hour})
	s2, _ := NewSigner(Config{Secret: "secret-2", UserTTL: time.Hour})
	tok, _ := s1.Issue(domain.Identity{UserID: "u1"}, time.Hour)
	if _, err := s2.Parse(tok.Value); err == nil {
		t.Error("expected parse failure with wrong secret")
	}
}

func TestSigner_TokenIncludesUserIDClaimForLegacyMiddleware(t *testing.T) {
	// Test the compatibility behavior: tokens carry both `sub` and
	// `user_id` so legacy middleware that reads `user_id` keeps working.
	s := newTestSigner(t)
	tok, _ := s.Issue(domain.Identity{UserID: "u-legacy"}, time.Hour)

	// Decode the JWT payload manually (no signature check) to verify
	// the `user_id` claim is present.
	parts := splitDots(tok.Value)
	if len(parts) != 3 {
		t.Fatal("malformed jwt")
	}
	payload := decodeJWTSegment(t, parts[1])
	if !contains(payload, `"user_id":"u-legacy"`) {
		t.Errorf("payload missing user_id claim: %s", payload)
	}
	if !contains(payload, `"sub":"u-legacy"`) {
		t.Errorf("payload missing sub claim: %s", payload)
	}
}

func TestTicketSigner_RoundTripPreservesScope(t *testing.T) {
	ts, err := NewTicketSigner(Config{Secret: "ticket-secret", TicketTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]string{"bot_id": "bot-1", "project_id": "p-1"}
	ticket, err := ts.Issue("u1", scope, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ts.Parse(ticket.Value)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.UserID != "u1" {
		t.Errorf("user_id = %q, want u1", parsed.UserID)
	}
	if parsed.Scope["bot_id"] != "bot-1" || parsed.Scope["project_id"] != "p-1" {
		t.Errorf("scope = %v", parsed.Scope)
	}
}

func TestTicketSigner_RejectsExpired(t *testing.T) {
	ts, _ := NewTicketSigner(Config{Secret: "s", TicketTTL: time.Minute})
	ticket, err := ts.Issue("u1", nil, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	_, err = ts.Parse(ticket.Value)
	if !errors.Is(err, domain.ErrInvalidWSTicket) {
		t.Errorf("expected ErrInvalidWSTicket, got %v", err)
	}
}

// --- helpers ----------------------------------------------------------

func splitDots(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func decodeJWTSegment(t *testing.T, seg string) string {
	t.Helper()
	// JWT uses URL-safe base64 without padding.
	pad := 4 - len(seg)%4
	if pad < 4 {
		seg += string(make([]byte, pad))
		for i := 0; i < pad; i++ {
			seg = seg[:len(seg)-1] + "="
		}
	}
	dec, err := base64DecodeURL(seg)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return string(dec)
}

// avoiding extra import for tests
func base64DecodeURL(s string) ([]byte, error) {
	// fall back to encoding/base64 lazily
	return decodeRawURL(s), nil
}

// decodeRawURL handles base64.RawURLEncoding manually to keep the test file
// import-light and predictable.
func decodeRawURL(s string) []byte {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	pos := make([]int, 256)
	for i := range pos {
		pos[i] = -1
	}
	for i, c := range alphabet {
		pos[c] = i
	}

	clean := []byte{}
	for _, c := range s {
		if c == '=' || c == ' ' {
			continue
		}
		clean = append(clean, byte(c))
	}
	out := make([]byte, 0, len(clean)*3/4)
	var buf uint32
	bits := 0
	for _, c := range clean {
		v := pos[c]
		if v < 0 {
			continue
		}
		buf = buf<<6 | uint32(v)
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(buf>>bits))
			buf &= (1 << bits) - 1
		}
	}
	return out
}

func contains(haystack, needle string) bool {
	return indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	n := len(needle)
	for i := 0; i+n <= len(haystack); i++ {
		if haystack[i:i+n] == needle {
			return i
		}
	}
	return -1
}
