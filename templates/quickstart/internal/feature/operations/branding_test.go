package operations

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSiteLogoPublishesOnlyTheSelectedAdministratorImage(t *testing.T) {
	f := newFixture(t)
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	imageID := func(owner string) string {
		t.Helper()
		w := uploadRequest(t, f, imageBytes.Bytes(), "admin", owner)
		if w.Code != 200 {
			t.Fatalf("upload: %s", w.Body)
		}
		return data(t, w)["id"].(string)
	}
	first, second, private := imageID("owner"), imageID("owner"), imageID("other")
	logoPath := func(id string) string { return "/site/logo/" + id }
	patch := func(id, key string) int {
		body, _ := json.Marshal(map[string]string{"logo_url": "/api/v1" + logoPath(id), "reason": "Update brand logo"})
		return request(f, "PATCH", "/admin/settings", string(body), "admin", key).Code
	}
	if w := request(f, "GET", logoPath(first), "", "", ""); w.Code != 404 {
		t.Fatalf("unpublished image was public: %d", w.Code)
	}
	if got := patch(private, "other-private-image"); got != 404 {
		t.Fatalf("another user's private image must stay private: %d", got)
	}
	if got := patch(first, "first-logo"); got != 200 {
		t.Fatalf("logo save: %d", got)
	}
	if got := patch(first, "first-logo"); got != 200 {
		t.Fatalf("idempotent logo save: %d", got)
	}
	for _, path := range []string{"/site/settings", "/admin/settings"} {
		if w := request(f, "GET", path, "", "admin", ""); data(t, w)["logo_url"] != "/api/v1"+logoPath(first) {
			t.Fatalf("logo missing from settings: %s", w.Body)
		}
	}
	w := request(f, "GET", logoPath(first), "", "", "")
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), imageBytes.Bytes()) || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("public logo response: %d %v", w.Code, w.Header())
	}
	// Another administrator can retain the already-public logo while editing settings.
	body, _ := json.Marshal(map[string]string{"logo_url": "/api/v1" + logoPath(first), "brand_name": "Updated brand", "reason": "Keep published logo"})
	r := httptest.NewRequest("PATCH", "/api/v1/admin/settings", bytes.NewReader(body))
	r.Header.Set("X-Test-Role", "admin")
	r.Header.Set("X-Test-User", "second-admin")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "retain-public-logo")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("retaining a published logo: %s", w.Body)
	}
	if got := patch(second, "second-logo"); got != 200 {
		t.Fatalf("replace logo: %d", got)
	}
	if w := request(f, "GET", logoPath(first), "", "", ""); w.Code != 404 {
		t.Fatal("replaced image remains public")
	}
	if w := request(f, "PATCH", "/admin/settings", `{"logo_url":"","reason":"Restore default logo"}`, "admin", "reset-logo"); w.Code != 200 {
		t.Fatalf("reset: %s", w.Body)
	}
	if w := request(f, "GET", logoPath(second), "", "", ""); w.Code != 404 {
		t.Fatal("reset image remains public")
	}
	var audits int64
	if err := f.db.Model(&AuditEvent{}).Where("action = ? AND status = ?", "settings.update", "succeeded").Count(&audits).Error; err != nil || audits != 4 {
		t.Fatalf("expected four audited changes, got %d: %v", audits, err)
	}
}

func TestSiteLogoURLValidation(t *testing.T) {
	f := newFixture(t)
	for _, value := range []string{"https://cdn.example.test/logo.svg", "/brand/logo.png", ""} {
		body, _ := json.Marshal(map[string]string{"logo_url": value, "reason": "Set hosted brand logo"})
		if w := request(f, "PATCH", "/admin/settings", string(body), "admin", "valid-"+value); w.Code != 200 {
			t.Fatalf("valid URL rejected %q: %s", value, w.Body)
		}
	}
	for i, value := range []string{"javascript:alert(1)", "data:image/svg+xml,x", "http://cdn.example.test/logo.png", "https://user:pass@example.test/logo.png", "//example.test/logo.png", "/\\example.test/logo.png", "/api/v1/uploads/images/private", "/api/v1/site/logo/not-a-uuid", strings.Repeat("x", 1025)} {
		body, _ := json.Marshal(map[string]string{"logo_url": value, "reason": "Reject unsafe logo"})
		if w := request(f, "PATCH", "/admin/settings", string(body), "admin", string(rune('a'+i))); w.Code != 400 {
			t.Fatalf("unsafe URL accepted %q: %d", value, w.Code)
		}
	}
}
