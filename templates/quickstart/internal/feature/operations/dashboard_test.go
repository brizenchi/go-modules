package operations

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/feature/membership"
	"github.com/brizenchi/quickstart-template/internal/user"
)

func TestOverviewCountsRealPaymentsOnceAndSeparatesTestMode(t *testing.T) {
	f := newFixture(t)
	for i, payload := range []string{
		`{"livemode":true,"data":{"object":{"id":"cs_live","invoice":"in_live","amount_total":900,"payment_status":"paid"}}}`,
		`{"livemode":true,"data":{"object":{"id":"in_live","amount_paid":900}}}`,
		`{"livemode":true,"data":{"object":{"id":"in_live","amount_paid":900}}}`,
		`{"livemode":false,"data":{"object":{"id":"cs_test","amount_total":900,"payment_status":"paid"}}}`,
		`{"livemode":true,"data":{"object":{"id":"cs_unpaid","amount_total":900,"payment_status":"unpaid"}}}`,
		`{"livemode":true,"data":{"object":{"id":"cs_zero","amount_total":0,"payment_status":"paid"}}}`,
	} {
		kind := "checkout.session.completed"
		if i == 1 {
			kind = "invoice.paid"
		}
		if i == 2 {
			kind = "invoice.payment_succeeded"
		}
		if err := f.db.Create(&billingdomain.BillingEvent{Provider: "stripe", ProviderEventID: fmt.Sprint("evt_", i), EventType: kind, UserID: "owner", Processed: true, Payload: json.RawMessage(payload)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"live", "test"} {
		w := request(f, "GET", "/admin/overview?timezone=Asia%2FShanghai&mode="+mode+"&days=7", "", "admin", "")
		body := data(t, w)
		if w.Code != 200 || body["today_payments"] != float64(1) || body["paid_users"] != float64(1) || body["today_registrations"] != float64(1) {
			t.Fatalf("overview %s: %s", mode, w.Body)
		}
		if len(body["series"].([]any)) != 7 {
			t.Fatal("missing daily series")
		}
	}
	for _, query := range []string{"days=9999", "mode=unknown", "timezone=bad"} {
		if w := request(f, "GET", "/admin/overview?"+query, "", "admin", ""); w.Code != 400 {
			t.Fatalf("invalid range accepted: %s", w.Body)
		}
	}
}

func TestUserMembershipEditsAreAuditedIdempotentAndDoNotChangeSecurityRole(t *testing.T) {
	f := newFixture(t)
	end := time.Now().UTC().AddDate(0, 1, 0)
	if err := f.db.Create(&billingdomain.BillingSubscription{UserID: "owner", Provider: "stripe", Plan: "premium", Status: "active", PeriodEnd: &end}).Error; err != nil {
		t.Fatal(err)
	}

	body := `{"plan":"pro","reason":"Support adjustment","expires_at":"` + time.Now().UTC().Add(24*time.Hour).Format(time.RFC3339) + `"}`
	for _, role := range []string{"", "user"} {
		w := request(f, "PATCH", "/admin/users/owner/membership", body, role, "grant")
		if w.Code != 401 && w.Code != 403 {
			t.Fatalf("unauthorized write: %s", w.Body)
		}
	}
	for i := 0; i < 2; i++ {
		w := request(f, "PATCH", "/admin/users/owner/membership", body, "admin", "grant")
		if w.Code != 200 {
			t.Fatalf("membership update: %d %s", w.Code, w.Body)
		}
	}
	var saved user.User
	if err := f.db.Where("id = ?", "owner").Take(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Role != "user" {
		t.Fatal("membership changed security role")
	}
	access, err := membership.Load(t.Context(), f.db, "owner")
	if err != nil || access.Plan != "pro" {
		t.Fatalf("effective membership: %+v %v", access, err)
	}
	var provider billingdomain.BillingSubscription
	if err := f.db.Where("user_id = ?", "owner").Take(&provider).Error; err != nil {
		t.Fatal(err)
	}
	if provider.Plan != "premium" {
		t.Fatal("manual membership altered provider subscription")
	}

	w := request(f, "GET", "/admin/users?query=owner", "", "admin", "")
	if !strings.Contains(w.Body.String(), `"plan":"pro"`) {
		t.Fatalf("user plan missing: %s", w.Body)
	}
	w = request(f, "PATCH", "/admin/users/owner/membership", strings.Replace(body, `"pro"`, `"premium"`, 1), "admin", "grant")
	if w.Code != 409 {
		t.Fatalf("conflicting key accepted: %s", w.Body)
	}
	var count int64
	f.db.Model(&AuditEvent{}).Where("action = ?", "membership.update").Count(&count)
	if count != 1 {
		t.Fatalf("audit count: %d", count)
	}
	if w := request(f, "PATCH", "/admin/users/missing/membership", body, "admin", "missing"); w.Code != 404 {
		t.Fatalf("missing user: %s", w.Body)
	}
	for _, invalid := range []string{`{"plan":"admin","reason":"Invalid plan"}`, `{"plan":"pro","reason":"Missing expiry"}`, `{"plan":"lifetime","reason":"x"}`, `{"plan":"lifetime","reason":"Invalid field","role":"admin"}`} {
		if w := request(f, "PATCH", "/admin/users/owner/membership", invalid, "admin", "invalid"); w.Code != 400 {
			t.Fatalf("invalid membership accepted: %s", w.Body)
		}
	}
	if w := request(f, "PATCH", "/admin/users/owner/membership", `{"plan":"automatic","reason":"Restore payment access"}`, "admin", "restore"); w.Code != 200 {
		t.Fatalf("restore: %s", w.Body)
	}
	access, err = membership.Load(t.Context(), f.db, "owner")
	if err != nil || access.Plan != "premium" {
		t.Fatalf("payment fallback: %+v %v", access, err)
	}

}

func TestAdminCreditDeductionKeepsLedgerAuditAndCannotOverdraw(t *testing.T) {
	f := newFixture(t)
	if err := user.NewRepository(f.db).GrantCredits(t.Context(), "owner", "test", "opening", 100); err != nil {
		t.Fatal(err)
	}
	body := `{"amount":30,"reason":"Support correction"}`
	for _, role := range []string{"", "user"} {
		if w := request(f, "POST", "/admin/users/owner/credit-deductions", body, role, "deduct-key"); w.Code != 401 && w.Code != 403 {
			t.Fatalf("unauthorized: %s", w.Body)
		}
	}
	for i := 0; i < 2; i++ {
		if w := request(f, "POST", "/admin/users/owner/credit-deductions", body, "admin", "deduct-key"); w.Code != 200 || data(t, w)["balance"] != float64(70) {
			t.Fatalf("deduct: %s", w.Body)
		}
	}
	if w := request(f, "POST", "/admin/users/owner/credit-deductions", `{"amount":31,"reason":"Support correction"}`, "admin", "deduct-key"); w.Code != 409 {
		t.Fatalf("changed retry: %s", w.Body)
	}
	if w := request(f, "POST", "/admin/users/owner/credit-deductions", `{"amount":71,"reason":"Support correction"}`, "admin", "overdraw-key"); w.Code != 409 {
		t.Fatalf("overdraw: %s", w.Body)
	}
	var entry user.CreditTransaction
	if err := f.db.Where("source = ?", "admin_deduct").Take(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.Amount != -30 || entry.ActorID != "owner" {
		t.Fatalf("ledger: %+v", entry)
	}
	var count int64
	f.db.Model(&AuditEvent{}).Where("action = ?", "credits.deduct").Count(&count)
	if count != 1 {
		t.Fatalf("audits: %d", count)
	}
}

func TestOverviewBucketsAtReportingMidnightAndExcludesFutureRecords(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 10, 15, 17, 0, 0, 0, time.UTC)
	if err := f.db.Model(&user.User{}).Where("id = ?", "owner").Update("created_at", now.AddDate(0, 0, -40)).Error; err != nil {
		t.Fatal(err)
	}
	// UTC+8 midnight is 16:00 UTC, an hour before the reporting instant.
	for i, stamp := range []time.Time{now.Add(-time.Hour - time.Second), now.Add(-time.Hour), now.Add(time.Minute)} {
		u := user.User{ID: fmt.Sprint("boundary_", i), Email: fmt.Sprintf("boundary%d@example.test", i), CreatedAt: stamp}
		if err := f.db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	got, err := loadOverview(t.Context(), f.db, now, time.FixedZone("UTC+8", 8*60*60), 7, "live", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got["today_registrations"] != int64(1) {
		t.Fatalf("wrong calendar counts: %+v", got)
	}
	series := got["series"].([]dailyMetrics)
	if series[6].Date != "2026-10-16" || series[5].Registrations != 1 {
		t.Fatalf("series: %+v", series)
	}
}

func TestAdminAccessDoesNotDependOnReportingDatabase(t *testing.T) {
	f := newFixture(t)
	f.module.deps.DB = nil
	w := request(f, "GET", "/admin/access", "", "admin", "")
	if w.Code != 200 || data(t, w)["authorized"] != true {
		t.Fatalf("access: %s", w.Body)
	}
}
