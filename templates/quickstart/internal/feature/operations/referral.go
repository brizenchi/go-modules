package operations

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	refdomain "github.com/brizenchi/go-modules/modules/referral/domain"
	refevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (m *Module) retryReward(c *gin.Context) {
	actor := userID(c)
	if actor == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		httpresp.BadRequest(c, "invalid referral ID")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeBody(c, &body) {
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	key, ok := operationMetadata(c, body.Reason)
	if !ok {
		return
	}
	var referral refdomain.Referral
	if err := db.Table("referrals").Where("id = ?", id).Take(&referral).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpresp.NotFound(c, "referral not found")
		} else {
			queryFailed(c, err)
		}
		return
	}
	// Operators can only replay a persisted qualifying event. Pending or expired
	// referrals can never be force-activated through this endpoint.
	if referral.Status != refdomain.StatusActivated || referral.ActivatedAt == nil || referral.RewardCredits < 0 {
		httpresp.Conflict(c, "only previously activated referrals can retry their stored reward")
		return
	}
	if m.deps.Modules == nil || m.deps.Modules.Referral == nil || m.deps.Modules.Referral.Deps.Bus == nil {
		httpresp.Custom(c, 503, 503, "referral delivery unavailable", nil)
		return
	}
	audit := AuditEvent{ActorID: actor, IdempotencyKey: key, Action: "referral.retry_reward", TargetID: strconv.FormatUint(id, 10), Reason: body.Reason, RequestHash: requestHash(body), Status: "pending"}
	err = db.Transaction(func(tx *gorm.DB) error { _, e := claimAudit(tx, &audit); return e })
	if errors.Is(err, errKeyConflict) {
		httpresp.Conflict(c, err.Error())
		return
	}
	if queryFailed(c, err) {
		return
	}
	if audit.Status != "succeeded" {
		err = m.deps.Modules.Referral.Deps.Bus.Publish(c.Request.Context(), refevent.Envelope{Kind: refevent.KindReferralActivated, OccurredAt: referral.ActivatedAt.UTC(), Payload: refevent.ReferralActivated{Referral: referral}})
		status, details := "succeeded", "Stored activation event delivered; credit grant remains idempotent."
		if err != nil {
			status, details = "failed", "Reward listener failed; retry with the same idempotency key."
		}
		// Do not store raw provider/listener errors (they can contain credentials).
		update := db.Model(&AuditEvent{}).Where("id = ?", audit.ID)
		if status == "failed" {
			// A concurrent successful retry is conclusive; a late failure must
			// not overwrite the successful audit outcome.
			update = update.Where("status <> ?", "succeeded")
		}
		if updateErr := update.Updates(map[string]any{"status": status, "details": details, "updated_at": time.Now().UTC()}).Error; updateErr != nil {
			queryFailed(c, updateErr)
			return
		}
		if err != nil {
			httpresp.Custom(c, 502, 502, details, nil)
			return
		}
	}
	httpresp.OK(c, gin.H{"referral_id": id, "status": "succeeded", "reward_credits": referral.RewardCredits, "audit_id": audit.ID})
}
