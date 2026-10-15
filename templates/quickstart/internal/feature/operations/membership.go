package operations

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/feature/membership"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errInvalidMembership = errors.New("invalid membership plan or expiry")

type memberPatch struct {
	Plan      string     `json:"plan"`
	ExpiresAt *time.Time `json:"expires_at"`
	Reason    string     `json:"reason"`
}

func validMembership(p memberPatch, now time.Time) bool {
	if p.ExpiresAt != nil && (!p.ExpiresAt.After(now) || p.ExpiresAt.After(now.AddDate(100, 0, 0))) {
		return false
	}
	switch p.Plan {
	case "automatic", "lifetime":
		return p.ExpiresAt == nil
	case "free":
		return true
	case "starter", "pro", "premium":
		return p.ExpiresAt != nil
	default:
		return false
	}
}
func (m *Module) userAccess(c *gin.Context) {
	db := m.db(c)
	if db == nil {
		return
	}
	var u user.User
	if err := db.Where("id = ?", c.Param("id")).Take(&u).Error; err != nil {
		respondMembershipError(c, err)
		return
	}
	states, err := membership.LoadStates(c.Request.Context(), db, []string{u.ID}, time.Now().UTC(), m.deps.Modules == nil || m.deps.Modules.Billing != nil)
	if queryFailed(c, err) {
		return
	}
	state := states[u.ID]
	if u.CreditsVersion == 1 && m.deps.Credits != nil {
		summary, err := m.deps.Credits.GetCreditSummary(c.Request.Context(), u.ID)
		if queryFailed(c, err) {
			return
		}
		u.Credits = summary.Balance
	}
	var subscription *subscriptionView
	if m.deps.Modules == nil || m.deps.Modules.Billing != nil {
		row := subscriptionView{}
		err := db.Table("billing_subscriptions").Where("user_id = ? AND provider = ?", u.ID, "stripe").Take(&row).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			queryFailed(c, err)
			return
		}
		if err == nil {
			subscription = &row
		}
	}
	httpresp.OK(c, gin.H{"membership": state.Membership, "billing": state.Billing, "override": state.Override, "user": u, "subscription": subscription})
}

func (m *Module) patchMembership(c *gin.Context) {
	actor := userID(c)
	if actor == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	var patch memberPatch
	if !decodeBody(c, &patch) {
		return
	}
	patch.Reason = strings.TrimSpace(patch.Reason)
	key, ok := operationMetadata(c, patch.Reason)
	if !ok {
		return
	}
	uid := c.Param("id")
	audit := AuditEvent{ActorID: actor, IdempotencyKey: key, Action: "membership.update", TargetID: uid, Reason: patch.Reason, RequestHash: requestHash(patch), Status: "succeeded"}
	var result struct {
		Before *membership.Override `json:"before"`
		After  *membership.Override `json:"after"`
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		created, err := claimAudit(tx, &audit)
		if err != nil {
			return err
		}
		if !created {
			return json.Unmarshal([]byte(audit.Details), &result)
		}
		if !validMembership(patch, time.Now().UTC()) {
			return errInvalidMembership
		}
		var u user.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", uid).Take(&u).Error; err != nil {
			return err
		}
		var before membership.Override
		err = tx.Where("user_id = ?", uid).Take(&before).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			result.Before = &before
		}
		if patch.Plan == "automatic" {
			if err := tx.Where("user_id = ?", uid).Delete(&membership.Override{}).Error; err != nil {
				return err
			}
		} else {
			row := membership.Override{UserID: uid, Plan: patch.Plan, ExpiresAt: patch.ExpiresAt, UpdatedAt: time.Now().UTC()}
			if row.ExpiresAt != nil {
				v := row.ExpiresAt.UTC()
				row.ExpiresAt = &v
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"plan", "expires_at", "updated_at"})}).Create(&row).Error; err != nil {
				return err
			}
			result.After = &row
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return m.deps.Audit.Finish(c.Request.Context(), tx, &audit, "succeeded", string(raw))
	})
	if err != nil {
		respondMembershipError(c, err)
		return
	}
	httpresp.OK(c, result)
}
func respondMembershipError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errInvalidMembership):
		httpresp.BadRequest(c, "invalid membership plan or expiry")
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpresp.NotFound(c, "account not found")
	case errors.Is(err, errKeyConflict):
		httpresp.Custom(c, 409, 409, "operation conflict", gin.H{"reason": "OPERATIONS_IDEMPOTENCY_CONFLICT"})
	default:
		queryFailed(c, err)
	}
}
