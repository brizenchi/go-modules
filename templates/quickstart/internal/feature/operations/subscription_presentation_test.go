package operations

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/feature/membership"
)

func TestAdminUsersExposeCurrentAndScheduledSubscriptionSeparately(t *testing.T) {
	f := newFixture(t)
	end := time.Now().UTC().AddDate(0, 1, 0).Truncate(time.Second)
	raw, err := json.Marshal(domain.SubscriptionSnapshot{Plan: domain.PlanPremium, Status: domain.StatusActive, ScheduledChange: &domain.ScheduledChange{Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, EffectiveAt: &end}})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&domain.BillingSubscription{UserID: "owner", Provider: "stripe", Plan: "premium", Status: "active", BillingInterval: "monthly", PeriodEnd: &end, RawSnapshotJSON: raw, ProviderCustomerID: "private-customer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&membership.Override{UserID: "owner", Plan: "pro", ExpiresAt: &end}).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin/users?query=owner", "/admin/users/owner/access", "/admin/subscriptions?query=owner"} {
		w := request(f, "GET", path, "", "admin", "")
		if w.Code != 200 {
			t.Fatalf("%s: %s", path, w.Body)
		}
		body := data(t, w)
		if items, ok := body["items"].([]any); ok {
			body = items[0].(map[string]any)
		}
		sub := body
		if !strings.HasPrefix(path, "/admin/subscriptions") {
			var ok bool
			sub, ok = body["subscription"].(map[string]any)
			if !ok {
				t.Fatalf("%s: missing subscription: %s", path, w.Body)
			}
			if body["membership"].(map[string]any)["plan"] != "pro" {
				t.Fatal("manual access confused with subscription")
			}
		}
		if sub["plan"] != "premium" {
			t.Fatalf("current plan replaced: %+v", sub)
		}
		schedule, ok := sub["scheduled_change"].(map[string]any)
		if !ok || schedule["plan"] != "starter" || schedule["billing_cycle"] != "monthly" || schedule["effective_at"] != end.Format(time.RFC3339) {
			t.Fatalf("%s: missing scheduled change: %+v", path, sub)
		}
		if strings.Contains(w.Body.String(), "RawSnapshot") || strings.Contains(w.Body.String(), "raw_snapshot") || strings.Contains(w.Body.String(), "private-customer") {
			t.Fatal("private snapshot leaked")
		}
	}
	for _, role := range []string{"", "user"} {
		w := request(f, "GET", "/admin/users/owner/access", "", role, "")
		if w.Code != 401 && w.Code != 403 {
			t.Fatalf("unauthorized: %d", w.Code)
		}
	}
}

func TestOperatorSubscriptionOmitsUnconfirmedOrObsoleteSchedules(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	future := now.AddDate(0, 1, 0)
	past := now.Add(-time.Hour)
	for _, tc := range []struct {
		name    string
		status  string
		cancel  bool
		pending bool
		at      *time.Time
		want    bool
	}{
		{"scheduled", "active", false, false, &future, true},
		{"canceled", "canceled", false, false, &future, false},
		{"canceling", "active", true, false, &future, false},
		{"payment pending", "active", false, true, &future, false},
		{"date passed", "active", false, false, &past, false},
		{"unknown date", "active", false, false, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := domain.SubscriptionSnapshot{ScheduledChange: &domain.ScheduledChange{Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, EffectiveAt: tc.at}}
			if tc.pending {
				snapshot.PendingPayment = &domain.PendingPayment{InvoiceURL: "https://example.test/private-invoice"}
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			got := operatorSubscription(&domain.BillingSubscription{Plan: "premium", Status: tc.status, CancelAtPeriodEnd: tc.cancel, RawSnapshotJSON: raw}, now)
			if (got.ScheduledChange != nil) != tc.want || got.Plan != "premium" || got.PaymentPending != tc.pending {
				t.Fatalf("unexpected presentation: %+v", got)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "private-invoice") {
				t.Fatal("payment URL leaked")
			}
		})
	}
	legacy := operatorSubscription(&domain.BillingSubscription{Plan: "premium", Status: "active", RawSnapshotJSON: json.RawMessage(`{"legacy":true}`)}, now)
	if legacy.ScheduledChange != nil || legacy.Plan != "premium" {
		t.Fatalf("legacy snapshot: %+v", legacy)
	}
	if operatorSubscription(nil, now) != nil {
		t.Fatal("missing subscription fabricated")
	}
}
