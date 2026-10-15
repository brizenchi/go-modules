package checkoutlog

import (
	"testing"
	"time"

	billingapp "github.com/brizenchi/go-modules/modules/billing/app"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCheckoutHistoryIsIdempotentAndDoesNotStorePaymentURLs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&Record{}); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	amount := int64(999)
	mode := false
	input := billingapp.CheckoutInput{UserID: "customer", ProductType: billingdomain.ProductCredits, Quantity: 2, SuccessURL: "https://app.example.test/secret-return"}
	result := billingapp.CheckoutResult{SessionID: "cs_history", CheckoutURL: "https://checkout.example.test/private-url", CreatedAt: &created, AmountTotal: &amount, Currency: "USD", Livemode: &mode}
	if err := Save(t.Context(), db, "stripe", input, result); err != nil {
		t.Fatal(err)
	}
	input.UserID = "other"
	input.Quantity = 20
	if err := Save(t.Context(), db, "stripe", input, result); err != nil {
		t.Fatal(err)
	}
	var rows []Record
	if err := db.Order("id ASC").Limit(10).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UserID != "customer" || rows[0].Quantity != 2 || rows[0].Currency != "usd" || !rows[0].CreatedAt.Equal(created) {
		t.Fatalf("replay rewrote original intent: %+v", rows)
	}
	if db.Migrator().HasColumn(&Record{}, "checkout_url") || db.Migrator().HasColumn(&Record{}, "success_url") {
		t.Fatal("payment/return URLs should not be stored")
	}
}
