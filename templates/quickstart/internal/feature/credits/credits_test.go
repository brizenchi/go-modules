package credits

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type creditHTTPFixture struct {
	router              *gin.Engine
	users               *user.Repository
	owner, other, admin *user.User
	note                *note.Note
	cost                int64
}

func newCreditHTTPFixture(t *testing.T) *creditHTTPFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "http.db")+"?_busy_timeout=10000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := user.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(append(Models(), &note.Note{})...); err != nil {
		t.Fatal(err)
	}
	f := &creditHTTPFixture{users: user.NewRepository(db), cost: 1}
	f.owner = &user.User{Email: "owner@example.com", Credits: 10}
	f.other = &user.User{Email: "other@example.com"}
	f.admin = &user.User{Email: "admin@example.com", Role: user.RoleAdmin}
	for _, account := range []*user.User{f.owner, f.other, f.admin} {
		if err := f.users.Create(context.Background(), account); err != nil {
			t.Fatal(err)
		}
	}
	f.note = &note.Note{UserID: f.owner.ID, Title: "My note", Body: "Original content"}
	if err := db.Create(f.note).Error; err != nil {
		t.Fatal(err)
	}
	f.router = gin.New()
	group := f.router.Group("/api/v1")
	group.Use(func(c *gin.Context) {
		var account *user.User
		switch c.GetHeader("X-Test-Identity") {
		case "owner":
			account = f.owner
		case "other":
			account = f.other
		case "admin":
			account = f.admin
		}
		if account != nil {
			authhttp.SetIdentity(c, &authdomain.Identity{UserID: account.ID, Role: authdomain.Role(account.Role)})
		}
		c.Next()
	})
	New(f.users, WithExportCost(func(context.Context) (int64, error) { return f.cost, nil })).Register(hostapi.Groups{User: group, Admin: group})
	return f
}
func (f *creditHTTPFixture) request(t *testing.T, method, path, actor string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var content []byte
	if body != nil {
		var err error
		content, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(content))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Identity", actor)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}
func expectHTTP(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status=%d want=%d body=%s", w.Code, status, w.Body.String())
	}
}
func decodeExport(t *testing.T, w *httptest.ResponseRecorder) ExportResult {
	t.Helper()
	expectHTTP(t, w, http.StatusOK)
	var body struct {
		Data ExportResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}

func TestPaidExportIsOwnedAtomicAndReplaysAfterPriceChange(t *testing.T) {
	f := newCreditHTTPFixture(t)
	path := fmt.Sprintf("/notes/%d/export", f.note.ID)
	first := decodeExport(t, f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "export-request-1", "expected_cost": 1}))
	if first.Balance != 9 || first.TransactionID == 0 || first.Content != "# My note\n\nOriginal content\n" {
		t.Fatalf("first=%+v", first)
	}
	f.cost = 4
	if err := f.users.DB().Model(&note.Note{}).Where("id = ?", f.note.ID).Update("body", "Changed content").Error; err != nil {
		t.Fatal(err)
	}
	replay := decodeExport(t, f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "export-request-1", "expected_cost": 1}))
	if replay != first {
		t.Fatalf("replay changed: %+v first=%+v", replay, first)
	}
	next := decodeExport(t, f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "export-request-2", "expected_cost": 4}))
	if next.Balance != 5 || next.TransactionID == first.TransactionID || next.Content == first.Content {
		t.Fatalf("new export=%+v", next)
	}
	expectHTTP(t, f.request(t, http.MethodPost, path, "other", gin.H{"idempotency_key": "export-request-1", "expected_cost": 1}), http.StatusNotFound)
	expectHTTP(t, f.request(t, http.MethodPost, path, "", gin.H{"idempotency_key": "export-request-1", "expected_cost": 1}), http.StatusUnauthorized)
	var count int64
	if err := f.users.DB().Model(&NoteExport{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("exports=%d", count)
	}
}

func TestPaidExportRejectsKeyReuseForOtherNoteAndInsufficientCredits(t *testing.T) {
	f := newCreditHTTPFixture(t)
	another := &note.Note{UserID: f.owner.ID, Title: "Another", Body: "Second"}
	if err := f.users.DB().Create(another).Error; err != nil {
		t.Fatal(err)
	}
	firstPath := fmt.Sprintf("/notes/%d/export", f.note.ID)
	secondPath := fmt.Sprintf("/notes/%d/export", another.ID)
	decodeExport(t, f.request(t, http.MethodPost, firstPath, "owner", gin.H{"idempotency_key": "same-export-key", "expected_cost": 1}))
	expectHTTP(t, f.request(t, http.MethodPost, secondPath, "owner", gin.H{"idempotency_key": "same-export-key", "expected_cost": 1}), http.StatusConflict)
	f.cost = 10
	expectHTTP(t, f.request(t, http.MethodPost, secondPath, "owner", gin.H{"idempotency_key": "too-expensive-key", "expected_cost": 10}), http.StatusConflict)
	summary, err := f.users.GetCreditSummary(context.Background(), f.owner.ID)
	if err != nil || summary.Balance != 9 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	var count int64
	if err := f.users.DB().Model(&NoteExport{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed exports persisted: %d", count)
	}
}

func TestAdminCreditWritesRequireRoleAndDeriveActor(t *testing.T) {
	f := newCreditHTTPFixture(t)
	grant := gin.H{"user_id": f.owner.ID, "amount": 8, "reason": "Support adjustment", "idempotency_key": "admin-grant-key"}
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/grants", "owner", grant), http.StatusForbidden)
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/grants", "", grant), http.StatusUnauthorized)
	grant["actor_id"] = f.owner.ID
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/grants", "admin", grant), http.StatusBadRequest)
	delete(grant, "actor_id")
	grant["role"] = "admin"
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/grants", "admin", grant), http.StatusBadRequest)
	delete(grant, "role")
	for i := 0; i < 2; i++ {
		expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/grants", "admin", grant), http.StatusOK)
	}
	var entries []user.CreditTransaction
	if err := f.users.DB().Where("source = ?", "admin_grant").Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ActorID != f.admin.ID || entries[0].Reason != "Support adjustment" {
		t.Fatalf("audit entries=%+v", entries)
	}
	summary, err := f.users.GetCreditSummary(context.Background(), f.owner.ID)
	if err != nil || summary.Balance != 18 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}

func TestAdminRefundRestoresOriginalUserAndCannotAcceptAmountOverride(t *testing.T) {
	f := newCreditHTTPFixture(t)
	exported := decodeExport(t, f.request(t, http.MethodPost, fmt.Sprintf("/notes/%d/export", f.note.ID), "owner", gin.H{"idempotency_key": "refund-export-key", "expected_cost": 1}))
	refund := gin.H{"transaction_id": exported.TransactionID, "reason": "Customer support refund", "idempotency_key": "admin-refund-key"}
	refund["amount"] = 999
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/refunds", "admin", refund), http.StatusBadRequest)
	delete(refund, "amount")
	for i := 0; i < 2; i++ {
		expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/refunds", "admin", refund), http.StatusOK)
	}
	refund["idempotency_key"] = "second-refund-key"
	expectHTTP(t, f.request(t, http.MethodPost, "/admin/credits/refunds", "admin", refund), http.StatusConflict)
	summary, err := f.users.GetCreditSummary(context.Background(), f.owner.ID)
	if err != nil || summary.Balance != 10 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}

func TestCreditHistoryCannotExposeAnotherUsersTransactions(t *testing.T) {
	f := newCreditHTTPFixture(t)
	w := f.request(t, http.MethodGet, "/credits/transactions?user_id="+f.owner.ID, "other", nil)
	expectHTTP(t, w, http.StatusOK)
	var body struct {
		Data user.CreditPage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Total != 0 || len(body.Data.List) != 0 {
		t.Fatalf("another user's records leaked: %+v", body.Data)
	}
	expectHTTP(t, f.request(t, http.MethodGet, "/admin/credits/transactions", "owner", nil), http.StatusForbidden)
	expectHTTP(t, f.request(t, http.MethodGet, "/credits/transactions?limit=101", "owner", nil), http.StatusBadRequest)
}

func TestExportRejectsUnconfirmedPriceWithoutCharging(t *testing.T) {
	f := newCreditHTTPFixture(t)
	path := fmt.Sprintf("/notes/%d/export", f.note.ID)
	f.cost = 5
	w := f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "confirmed-export-key", "expected_cost": 1})
	expectHTTP(t, w, http.StatusConflict)
	var errorBody struct {
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errorBody); err != nil {
		t.Fatal(err)
	}
	if errorBody.Msg != "price_changed" {
		t.Fatalf("wrong conflict: %s", w.Body.String())
	}
	expectHTTP(t, f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "confirmed-export-key"}), http.StatusBadRequest)
	summary, err := f.users.GetCreditSummary(context.Background(), f.owner.ID)
	if err != nil || summary.Balance != 10 {
		t.Fatalf("unexpected charge: %+v err=%v", summary, err)
	}
	confirmed := decodeExport(t, f.request(t, http.MethodPost, path, "owner", gin.H{"idempotency_key": "confirmed-export-key", "expected_cost": 5}))
	if confirmed.Balance != 5 {
		t.Fatalf("confirmed export=%+v", confirmed)
	}
}
