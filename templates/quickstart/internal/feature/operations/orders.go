package operations

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/gin-gonic/gin"
)

// OrderSummary is projected from verified, persisted provider events. Amount is
// in the provider currency's smallest unit. Missing amounts remain null.
type OrderSummary struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Email           string    `json:"email"`
	Provider        string    `json:"provider"`
	ProviderEventID string    `json:"provider_event_id"`
	EventType       string    `json:"event_type"`
	RecordType      string    `json:"record_type"`
	Status          string    `json:"status"`
	Amount          *int64    `json:"amount"`
	Currency        string    `json:"currency"`
	Processed       bool      `json:"processed"`
	Livemode        *bool     `json:"livemode"`
	CreatedAt       time.Time `json:"created_at"`
	sortID          uint
	eventAt         int64
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
	ID            string          `json:"id"`
	Invoice       json.RawMessage `json:"invoice"`
	AmountPaid    *int64          `json:"amount_paid"`
	AmountTotal   *int64          `json:"amount_total"`
	Currency      string          `json:"currency"`
	PaymentStatus string          `json:"payment_status"`
	Status        string          `json:"status"`
	Created       int64           `json:"created"`
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
	result.eventAt = envelope.Created
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

func (m *Module) orders(c *gin.Context) {
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
	byID := map[string]OrderSummary{}
	lastID := uint(0)
	for {
		batch := []billingdomain.BillingEvent{}
		err := db.Where("id > ? AND event_type IN ?", lastID, []string{"invoice.paid", "invoice.payment_succeeded", "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed"}).Order("id ASC").Limit(500).Find(&batch).Error
		if queryFailed(c, err) {
			return
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
			if exists && !replaceOrder(previous, item) {
				continue
			}
			byID[item.ID] = item
		}
	}
	identities := []struct {
		ID    string
		Email string
	}{}
	if queryFailed(c, db.Table("users").Select("id,email").Find(&identities).Error) {
		return
	}
	emails := map[string]string{}
	for _, identity := range identities {
		emails[identity.ID] = identity.Email
	}
	items := make([]OrderSummary, 0, len(byID))
	term := strings.ToLower(p.query)
	for _, item := range byID {
		item.Email = emails[item.UserID]
		if term != "" && !strings.Contains(strings.ToLower(item.ID+" "+item.UserID+" "+item.Email+" "+item.ProviderEventID), term) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
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
	httpresp.OK(c, gin.H{"items": items[start:end], "total": total, "page": p.page, "limit": p.limit, "source": "provider_payment_events", "amount_unit": "minor"})
}
