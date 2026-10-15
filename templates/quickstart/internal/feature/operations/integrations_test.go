package operations

import (
	"context"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"strings"
	"testing"
)

func TestProviderCredentialsAreReadOnlyAndNeverReturned(t *testing.T) {
	f := newFixture(t)
	off := false
	f.module.deps.ServiceSettings = serviceconfig.NewManager(f.db, platform.Config{Billing: platform.BillingConfig{Enabled: &off, Stripe: platform.StripeConfig{SecretKey: "sk_test_not-a-real-key", WebhookSecret: "whsec_not-a-real-secret"}}}, true)
	if _, err := f.module.deps.ServiceSettings.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"", "user"} {
		if w := request(f, "GET", "/admin/integrations", "", role, ""); w.Code != 401 && w.Code != 403 {
			t.Fatalf("permission: %s", w.Body)
		}
	}
	w := request(f, "GET", "/admin/integrations", "", "admin", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "not-a-real") {
		t.Fatalf("unsafe snapshot: %s", w.Body)
	}
	w = request(f, "PATCH", "/admin/integrations/stripe", `{"secrets":{"secret_key":"sk_test_not-a-real-key"},"reason":"Replace key"}`, "admin", "replace-key")
	if w.Code != 403 || !strings.Contains(w.Body.String(), "INTEGRATION_DEPLOYMENT_MANAGED") {
		t.Fatalf("credential write accepted: %s", w.Body)
	}
	var count int64
	f.db.Model(&serviceconfig.Record{}).Count(&count)
	if count != 0 {
		t.Fatal("credential boundary wrote configuration")
	}
}
