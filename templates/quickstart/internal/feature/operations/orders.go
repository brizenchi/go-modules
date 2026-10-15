package operations

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrderSummary combines recorded checkout intent with verified provider events
// for administrators. Customer purchases use verified events only. Amounts are
// in the provider currency's smallest unit; unknown amounts remain null.
type OrderSummary struct {
	CheckoutSessionID string     `json:"checkout_session_id,omitempty"`
	Plan              string     `json:"plan,omitempty"`
	BillingInterval   string     `json:"billing_interval,omitempty"`
	Quantity          int64      `json:"quantity,omitempty"`
	AmountDue         *int64     `json:"amount_due"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	ProductType       string     `json:"product_type,omitempty"`
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Email             string     `json:"email"`
	Provider          string     `json:"provider"`
	ProviderEventID   string     `json:"provider_event_id"`
	EventType         string     `json:"event_type"`
	RecordType        string     `json:"record_type"`
	Status            string     `json:"status"`
	Amount            *int64     `json:"amount"`
	Currency          string     `json:"currency"`
	Processed         bool       `json:"processed"`
	Livemode          *bool      `json:"livemode"`
	CreatedAt         time.Time  `json:"created_at"`
	sortID            uint
	eventAt           int64
	paidAt            time.Time
}

func objectID(raw json.RawMessage) string {
	var id string
	if json.Unmarshal(raw, &id) == nil {
		return id
	}
	var object struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &object)
	return object.ID
}

type paymentEnvelope struct {
	Livemode *bool `json:"livemode"`
	Created  int64 `json:"created"`
	Data     struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}
type paymentObject struct {
	Metadata          map[string]string `json:"metadata"`
	Mode              string            `json:"mode"`
	ID                string            `json:"id"`
	Invoice           json.RawMessage   `json:"invoice"`
	AmountPaid        *int64            `json:"amount_paid"`
	AmountTotal       *int64            `json:"amount_total"`
	Currency          string            `json:"currency"`
	PaymentStatus     string            `json:"payment_status"`
	Status            string            `json:"status"`
	Created           int64             `json:"created"`
	StatusTransitions struct {
		PaidAt int64 `json:"paid_at"`
	} `json:"status_transitions"`
}

func summarizeOrder(e billingdomain.BillingEvent) (OrderSummary, bool) {
	var envelope paymentEnvelope
	if json.Unmarshal(e.Payload, &envelope) != nil {
		envelope = paymentEnvelope{}
	}
	var object paymentObject
	if json.Unmarshal(envelope.Data.Object, &object) != nil {
		object = paymentObject{}
	}
	result := OrderSummary{UserID: e.UserID, Provider: e.Provider, ProviderEventID: e.ProviderEventID, EventType: e.EventType, Processed: e.Processed, Status: "unknown", Currency: strings.ToLower(object.Currency), Livemode: envelope.Livemode, CreatedAt: e.CreatedAt, sortID: e.ID}
	result.ProductType = object.Metadata["product_type"]
	if result.ProductType == "" && object.Mode == "subscription" {
		result.ProductType = "subscription"
	}
	result.eventAt = envelope.Created
	result.paidAt = e.CreatedAt
	if envelope.Created > 0 {
		result.paidAt = time.Unix(envelope.Created, 0).UTC()
	}
	if object.StatusTransitions.PaidAt > 0 {
		result.paidAt = time.Unix(object.StatusTransitions.PaidAt, 0).UTC()
	}
	if object.Created > 0 {
		result.CreatedAt = time.Unix(object.Created, 0).UTC()
	}
	id := object.ID
	switch e.EventType {
	case "invoice.paid", "invoice.payment_succeeded":
		result.RecordType = "invoice"
		result.Status = "paid"
		result.Amount = object.AmountPaid
	case "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed":
		result.CheckoutSessionID = object.ID
		result.RecordType = "checkout"
		result.Amount = object.AmountTotal
		if invoice := objectID(object.Invoice); invoice != "" {
			id = invoice
			result.RecordType = "invoice"
		}
		result.Status = object.PaymentStatus
		if result.Status == "" {
			result.Status = "unknown"
		}
		if e.EventType == "checkout.session.async_payment_succeeded" {
			result.Status = "paid"
		}
		if e.EventType == "checkout.session.async_payment_failed" {
			result.Status = "failed"
		}
	default:
		return OrderSummary{}, false
	}
	if id == "" {
		id = e.ProviderEventID
		result.RecordType = "payment_event"
	}
	result.AmountDue = result.Amount
	if result.Status == "paid" {
		paid := result.paidAt
		result.PaidAt = &paid
	}
	result.ID = e.Provider + ":" + id
	if result.Amount != nil && *result.Amount < 0 {
		result.Amount = nil
	}
	return result, true
}

func replaceOrder(previous, incoming OrderSummary) bool {
	previousInvoice := strings.HasPrefix(previous.EventType, "invoice.")
	incomingInvoice := strings.HasPrefix(incoming.EventType, "invoice.")
	if previousInvoice != incomingInvoice {
		return incomingInvoice
	}
	// Payment outcome events take precedence over checkout's initial completion
	// even when an old event is delivered after the payment outcome.
	previousFinal := previous.EventType == "checkout.session.async_payment_succeeded" || previous.EventType == "checkout.session.async_payment_failed"
	incomingFinal := incoming.EventType == "checkout.session.async_payment_succeeded" || incoming.EventType == "checkout.session.async_payment_failed"
	if previousFinal != incomingFinal {
		return incomingFinal
	}
	if previous.eventAt != 0 && incoming.eventAt != 0 && previous.eventAt != incoming.eventAt {
		return incoming.eventAt > previous.eventAt
	}
	// Equal or absent event timestamps cannot overturn confirmed payment with
	// an inconclusive older state. Prefer a populated amount when either copy
	// of an invoice contains additional verified data.
	if previous.Status == "paid" && incoming.Status != "paid" {
		return false
	}
	if previous.Amount != nil && incoming.Amount == nil {
		return false
	}
	return true
}

func (m *Module) purchases(c *gin.Context) {
	owner := userID(c)
	if owner == "" {
		return
	}
	m.paymentHistory(c, owner)
}

func (m *Module) orders(c *gin.Context) { m.paymentHistory(c, "") }

func (m *Module) paymentHistory(c *gin.Context, owner string) {
	includeIntents := owner == ""
	p, ok := pagination(c)
	if !ok {
		return
	}
	if m.deps.Modules != nil && m.deps.Modules.Billing == nil {
		pageResponse(c, p, []OrderSummary{}, 0)
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	// Stream event batches to avoid holding webhook payloads in memory. The
	// minimal template derives summaries on demand; larger installations should
	// materialize this projection from their billing listener.
	if owner == "" {
		owner = strings.TrimSpace(c.Query("user_id"))
	}
	if owner != "" {
		db = db.Where("user_id = ?", owner)
	}
	byID, err := loadOrderSummaries(c.Request.Context(), db)
	if queryFailed(c, err) {
		return
	}
	if includeIntents {
		if queryFailed(c, mergeCheckoutHistory(c.Request.Context(), db, byID)) {
			return
		}
	}
	identities := []struct {
		ID    string
		Email string
	}{}
	identityQuery := m.db(c).Table("users").Select("id,email")
	if owner != "" {
		identityQuery = identityQuery.Where("id = ?", owner)
	}
	if queryFailed(c, identityQuery.Order("id ASC").Limit(10000).Find(&identities).Error) {
		return
	}
	emails := map[string]string{}
	for _, identity := range identities {
		emails[identity.ID] = identity.Email
	}
	var from, to time.Time
	for name, target := range map[string]*time.Time{"from": &from, "to": &to} {
		if raw := c.Query(name); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				httpresp.BadRequest(c, "invalid payment date range")
				return
			}
			*target = parsed
		}
	}
	if !from.IsZero() && !to.IsZero() && !from.Before(to) {
		httpresp.BadRequest(c, "invalid payment date range")
		return
	}
	mode := c.Query("mode")
	if mode != "" && mode != "live" && mode != "test" {
		httpresp.BadRequest(c, "invalid payment environment")
		return
	}
	state := c.Query("status")
	if state != "" && state != "paid" && state != "unpaid" && state != "failed" && state != "unknown" && state != "no_payment_required" && state != "expired" {
		httpresp.BadRequest(c, "invalid payment status")
		return
	}
	items := make([]OrderSummary, 0, len(byID))
	term := strings.ToLower(p.query)
	for _, item := range byID {
		if mode != "" && (item.Livemode == nil || *item.Livemode != (mode == "live")) {
			continue
		}
		if state != "" && item.Status != state {
			continue
		}
		if !from.IsZero() && item.CreatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && !item.CreatedAt.Before(to) {
			continue
		}
		item.Email = emails[item.UserID]
		if term != "" && !strings.Contains(strings.ToLower(item.ID+" "+item.UserID+" "+item.Email+" "+item.ProviderEventID), term) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			if items[i].sortID == items[j].sortID {
				return items[i].ID > items[j].ID
			}
			return items[i].sortID > items[j].sortID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := len(items)
	start := (p.page - 1) * p.limit
	if start > total {
		start = total
	}
	end := start + p.limit
	if end > total {
		end = total
	}
	source := "provider_payment_events"
	if includeIntents {
		source = "checkout_history_and_provider_events"
	}
	httpresp.OK(c, gin.H{"items": items[start:end], "total": total, "page": p.page, "limit": p.limit, "source": source, "amount_unit": "minor"})
}

// loadOrderSummaries shares the verified, deduplicated payment projection with reporting.
func loadOrderSummaries(ctx context.Context, db *gorm.DB) (map[string]OrderSummary, error) {
	db = db.WithContext(ctx)
	byID := map[string]OrderSummary{}
	lastID := uint(0)
	for {
		batch := []billingdomain.BillingEvent{}
		err := db.Where("id > ? AND event_type IN ?", lastID, []string{"invoice.paid", "invoice.payment_succeeded", "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed"}).Order("id ASC").Limit(500).Find(&batch).Error
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, event := range batch {
			lastID = event.ID
			item, valid := summarizeOrder(event)
			if !valid {
				continue
			}
			previous, exists := byID[item.ID]
			// An invoice is the authoritative monetary record if checkout also
			// references it. Repeated provider event kinds stay one purchase.
			if exists {
				if item.CheckoutSessionID == "" {
					item.CheckoutSessionID = previous.CheckoutSessionID
				}
				if item.ProductType == "" {
					item.ProductType = previous.ProductType
				}
				if !replaceOrder(previous, item) {
					if previous.CheckoutSessionID == "" {
						previous.CheckoutSessionID = item.CheckoutSessionID
						byID[item.ID] = previous
					}
					if previous.ProductType == "" {
						previous.ProductType = item.ProductType
						byID[item.ID] = previous
					}
					continue
				}
			}
			byID[item.ID] = item
		}
	}
	return byID, nil
}
