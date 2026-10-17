package bootstrap

import (
	"fmt"
	"sync"
	"testing"

	"encoding/json"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreditsPurchaseDifferentNotificationsGrantTheCheckoutOnce(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint("legacy-", legacy), func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, _ := db.DB()
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err := user.AutoMigrate(db); err != nil {
				t.Fatal(err)
			}
			if err := db.AutoMigrate(&billingdomain.BillingEvent{}); err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"evt_completed", "evt_async_succeeded"} {
				if err := db.Create(&billingdomain.BillingEvent{UserID: "buyer", Provider: "stripe", ProviderEventID: id, EventType: "checkout.session.completed", Payload: json.RawMessage(`{"data":{"object":{"id":"cs_purchase","payment_status":"paid"}}}`)}).Error; err != nil {
					t.Fatal(err)
				}
			}
			users := user.NewRepository(db)
			if err := users.Create(t.Context(), &user.User{ID: "buyer", Email: "buyer@example.test"}); err != nil {
				t.Fatal(err)
			}
			deps := hostapi.Deps{DB: db, Users: users}.WithServices()
			if legacy {
				if err := deps.Credits.GrantCredits(t.Context(), "buyer", "stripe", "evt_completed", 100); err != nil {
					t.Fatal(err)
				}
			}
			purchase := billingevent.CreditsPurchased{CheckoutSessionID: "cs_purchase", Quantity: 1, CreditsPerUnit: 100, TotalCredits: 100, PriceID: "price_example"}
			for _, id := range []string{"evt_completed", "evt_async_succeeded", "evt_completed"} {
				if err := onCreditsPurchased(t.Context(), deps, billingevent.Envelope{UserID: "buyer", Provider: "stripe", ProviderEventID: id}, purchase); err != nil {
					t.Fatal(err)
				}
			}
			var workers sync.WaitGroup
			for i := range 8 {
				workers.Add(1)
				go func() {
					defer workers.Done()
					if err := onCreditsPurchased(t.Context(), deps, billingevent.Envelope{UserID: "buyer", Provider: "stripe", ProviderEventID: fmt.Sprint("evt_concurrent_", i)}, purchase); err != nil {
						t.Error(err)
					}
				}()
			}
			workers.Wait()
			summary, err := deps.Credits.GetCreditSummary(t.Context(), "buyer")
			if err != nil || summary.Balance != 100 {
				t.Fatalf("balance=%d err=%v; one checkout must grant once", summary.Balance, err)
			}
		})
	}
}
