package membership

import (
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEvaluate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Second)
	for _, tc := range []struct {
		name, plan, status string
		end, cancel        *time.Time
		active             bool
	}{
		{"active", "pro", "active", &future, nil, true},
		{"expiry boundary", "pro", "active", &now, nil, false},
		{"late webhook", "pro", "active", &past, nil, false},
		{"cancel retains access", "pro", "canceling", &future, &future, true},
		{"custom cancel elapsed", "pro", "canceling", &future, &past, false},
		{"failed payment", "pro", "past_due", &future, nil, false},
		{"missing period", "pro", "active", nil, nil, false},
		{"lifetime", "lifetime", "active", nil, nil, true},
		{"revoked lifetime", "lifetime", "canceled", nil, nil, false},
		{"unknown plan", "unknown", "active", &future, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := Evaluate(domain.BillingSubscription{Plan: tc.plan, Status: tc.status, PeriodEnd: tc.end, CancelEffectiveAt: tc.cancel}, now)
			if v.Active != tc.active {
				t.Fatalf("%+v", v)
			}
		})
	}
}

func TestRequireChecksOwnerAndMinimumTier(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&domain.BillingSubscription{}); err != nil {
		t.Fatal(err)
	}
	end := time.Now().UTC().Add(time.Hour)
	for _, row := range []domain.BillingSubscription{{UserID: "owner", Provider: "stripe", Plan: "starter", Status: "active", PeriodEnd: &end}, {UserID: "other", Provider: "stripe", Plan: "lifetime", Status: "active"}} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		user, minimum string
		status        int
	}{{"owner", "starter", 200}, {"owner", "pro", 403}, {"other", "premium", 200}, {"", "starter", 401}, {"other", "typo", 403}} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if tc.user != "" {
				authhttp.SetIdentity(c, &authdomain.Identity{UserID: tc.user})
			}
		})
		r.GET("/paid", Require(db, tc.minimum), func(c *gin.Context) { c.Status(200) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/paid?user_id=other", nil))
		if w.Code != tc.status {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body)
		}
	}
}
