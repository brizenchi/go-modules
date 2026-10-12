package billingrecovery

import (
	"context"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing"
	"github.com/brizenchi/go-modules/modules/billing/adapter/repo"
	"github.com/brizenchi/go-modules/modules/billing/app"
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type provider struct {
	port.Provider
	calls  int
	cursor string
	more   bool
}

func (p *provider) Name() string  { return "stripe" }
func (p *provider) Enabled() bool { return true }
func (p *provider) ListRecoveryEvents(_ context.Context, _, _ time.Time, after string) ([]string, bool, error) {
	p.calls++
	p.cursor = after
	return []string{"evt_test"}, p.more, nil
}
func (p *provider) RetrieveEvent(_ context.Context, id string) (*port.WebhookParseResult, error) {
	return &port.WebhookParseResult{ProviderEventID: id, Type: "test", RawPayload: []byte(`{}`)}, nil
}

func TestRecoveryLeaseCursorAndIdempotentReplay(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&State{}, &domain.BillingEvent{}, &domain.BillingSubscription{}); err != nil {
		t.Fatal(err)
	}
	p := &provider{more: true}
	events := repo.NewBillingEventRepo(db)
	m := &billing.Module{Provider: p, Webhook: app.NewWebhookService(p, events, nil, nil, nil)}
	ctx := context.Background()
	if err := Run(ctx, db, m); err != nil {
		t.Fatal(err)
	}
	var state State
	if err := db.First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Cursor != "evt_test" || state.Until.IsZero() {
		t.Fatalf("pagination not saved: %+v", state)
	}
	if err := db.Model(&State{}).Where("id = ?", "stripe").Update("lease_until", time.Now().UTC().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, db, m); err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 {
		t.Fatal("another worker ignored lease")
	}
	if err := db.Model(&State{}).Where("id = ?", "stripe").Update("lease_until", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	p.more = false
	if err := Run(ctx, db, m); err != nil {
		t.Fatal(err)
	}
	if p.cursor != "evt_test" {
		t.Fatal("cursor lost")
	}
	var count int64
	db.Model(&domain.BillingEvent{}).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate event rows: %d", count)
	}
	if err := db.First(&state).Error; err != nil {
		t.Fatal(err)
	}
	if state.Cursor != "" || !state.Until.IsZero() {
		t.Fatalf("window not completed: %+v", state)
	}
}
