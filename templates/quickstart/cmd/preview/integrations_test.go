package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewServiceSettingsStayIsolatedAfterSavingEnabledProvider(t *testing.T) {
	app, err := newPreview("http://localhost:3100", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	admin := previewLogin(t, app, previewAdmin, "")
	status := previewData(t, previewRequest(t, app, "GET", "/admin/integrations", admin, nil))
	if status["local_preview"] != true || status["storage_ready"] != true {
		t.Fatal("preview service settings must report temporary local storage")
	}
	const secret = "sk_test_not-a-real-key"
	payload, err := json.Marshal(map[string]any{
		"version": 0, "source": "database", "enabled": true,
		"fields":  map[string]string{"mode": "test", "pro_monthly": "price_IsolatedPreview123456789", "trial_days": "0", "credits_per_package": "100"},
		"secrets": map[string]string{"secret_key": secret, "webhook_secret": "whsec_not-a-real-secret"},
		"reason":  "Local form verification",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/integrations/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+admin)
	req.Header.Set("Idempotency-Key", "isolated-service-settings-save")
	w := httptest.NewRecorder()
	app.handler.ServeHTTP(w, req)
	saved := previewData(t, w)
	if saved["source"] != "database" || saved["enabled"] != true || saved["active_enabled"] != false {
		t.Fatal("saving preview configuration changed its active service state")
	}
	if strings.Contains(w.Body.String(), secret) {
		t.Fatal("secret returned by save response")
	}
	if app.modules.Billing != nil || app.modules.Config.Email.Provider != "log" {
		t.Fatal("saving service settings activated an external fixture provider")
	}
	capabilities := previewData(t, previewRequest(t, app, "GET", "/capabilities", "", nil))
	encoded, err := json.Marshal(capabilities)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatal("secret returned by public capabilities")
	}
	if response := previewRequest(t, app, "POST", "/stripe/checkout/session", admin, map[string]any{}); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("preview payment became available: %d", response.Code)
	}
}
