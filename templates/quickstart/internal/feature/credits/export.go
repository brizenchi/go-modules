package credits

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ExportResult struct {
	Filename      string `json:"filename"`
	Content       string `json:"content"`
	TransactionID uint64 `json:"transaction_id"`
	Balance       int64  `json:"balance"`
}

var ErrExportPriceChanged = errors.New("price_changed")

func (m *Module) exportNote(c *gin.Context) {
	id := identity(c, false)
	if id == nil {
		return
	}
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || noteID == 0 {
		httpresp.BadRequest(c, "invalid note id")
		return
	}
	var req struct {
		IdempotencyKey string `json:"idempotency_key"`
		ExpectedCost   int64  `json:"expected_cost"`
	}
	if !decode(c, &req) {
		return
	}
	if !validRequestKey(req.IdempotencyKey) || req.ExpectedCost < 1 || req.ExpectedCost > 1_000_000 {
		httpresp.BadRequest(c, "expected_cost (1–1000000) and idempotency_key (8–128 characters) are required")
		return
	}
	result, err := m.ExportNote(c.Request.Context(), id.UserID, noteID, req.IdempotencyKey, req.ExpectedCost)
	if err != nil {
		failure(c, err)
		return
	}
	httpresp.OK(c, result)
}

// ExportNote stores a snapshot in the same transaction as the charge. Repeating
// a successful request returns that snapshot even after the note or price changes.
func (m *Module) ExportNote(ctx context.Context, userID string, noteID uint64, key string, expectedCost int64) (*ExportResult, error) {
	if !validRequestKey(key) || noteID == 0 || expectedCost < 1 || expectedCost > 1_000_000 {
		return nil, credit.ErrInvalidOperation
	}
	repository := exportRepository{db: m.db}
	lookup := func() (*NoteExport, error) { return repository.find(ctx, userID, key, noteID) }
	result := func(saved *NoteExport) (*ExportResult, error) {
		summary, err := m.service.GetCreditSummary(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &ExportResult{Filename: saved.Filename, Content: saved.Content, TransactionID: saved.TransactionID, Balance: summary.Balance}, nil
	}
	saved, err := lookup()
	if err == nil {
		return result(saved)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// Ownership is checked before revealing an insufficient balance or charging.
	if err := repository.checkOwner(ctx, userID, noteID); err != nil {
		return nil, err
	}
	cost, err := m.exportCost(ctx)
	if err != nil {
		return nil, err
	}
	if cost < 1 || cost > 1_000_000 {
		return nil, credit.ErrInvalidOperation
	}
	if cost != expectedCost {
		return nil, ErrExportPriceChanged
	}
	_, err = m.service.ConsumeCreditsAndDo(ctx, credit.Consumption{UserID: userID, Source: "note_export", SourceID: userID + ":" + key, Amount: cost, Reason: fmt.Sprintf("Export note %d", noteID)}, func(tx *gorm.DB, entry *credit.Transaction) error {
		return repository.saveInTransaction(ctx, tx, userID, key, noteID, entry)
	})
	if err != nil && !errors.Is(err, credit.ErrConflict) {
		return nil, err
	}
	// Another request with the same key may have committed at a different price;
	// its stored result is authoritative and does not incur a second charge.
	saved, lookupErr := lookup()
	if lookupErr != nil {
		if err != nil {
			return nil, err
		}
		return nil, lookupErr
	}
	return result(saved)
}
