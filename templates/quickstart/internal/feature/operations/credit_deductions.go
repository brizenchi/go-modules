package operations

import (
	"errors"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (m *Module) deductCredits(c *gin.Context) {
	actor := userID(c)
	if actor == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	var req struct {
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if !decodeBody(c, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	key, ok := operationMetadata(c, req.Reason)
	if !ok {
		return
	}
	if req.Amount <= 0 || req.Amount > user.MaxCreditAmount {
		httpresp.BadRequest(c, "invalid credit amount")
		return
	}
	service := m.deps.Credits
	if service == nil {
		service = credit.New(user.NewRepository(db))
	}
	uid := c.Param("id")
	entry, err := service.ConsumeCreditsAndDo(c.Request.Context(), credit.Consumption{UserID: uid, Amount: req.Amount, Source: "admin_deduct", SourceID: actor + ":" + key, Reason: req.Reason}, func(tx *gorm.DB, entry *credit.Transaction) error {
		audit := AuditEvent{ActorID: actor, IdempotencyKey: key, Action: "credits.deduct", TargetID: uid, Reason: req.Reason, RequestHash: requestHash(req), Status: "succeeded"}
		if _, err := claimAudit(tx, &audit); err != nil {
			return err
		}
		entry.ActorID = actor
		if err := m.deps.Audit.Finish(c.Request.Context(), tx, &audit, "succeeded", stringJSON(map[string]any{"before": entry.BalanceAfter - entry.Amount, "after": entry.BalanceAfter, "amount": entry.Amount, "transaction_id": entry.ID})); err != nil {
			return err
		}
		return tx.Model(&user.CreditTransaction{}).Where("id = ?", entry.ID).Update("actor_id", actor).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, credit.ErrInsufficientCredits):
			httpresp.Custom(c, 409, 409, "insufficient credits", gin.H{"reason": "CREDITS_INSUFFICIENT_BALANCE"})
		case errors.Is(err, credit.ErrConflict):
			httpresp.Custom(c, 409, 409, "operation conflict", gin.H{"reason": "OPERATIONS_IDEMPOTENCY_CONFLICT"})
		case errors.Is(err, credit.ErrMigrationRequired):
			httpresp.Custom(c, 503, 503, "credit ledger unavailable", gin.H{"reason": "CREDITS_MIGRATION_REQUIRED"})
		case errors.Is(err, credit.ErrInvalidOperation):
			httpresp.BadRequest(c, "invalid credit operation")
		default:
			respondMembershipError(c, err)
		}
		return
	}
	balance, err := service.GetCreditSummary(c.Request.Context(), uid)
	if queryFailed(c, err) {
		return
	}
	httpresp.OK(c, gin.H{"transaction": entry, "balance": balance.Balance})
}
