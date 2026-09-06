package operations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errKeyConflict = errors.New("idempotency key already used for another request")

func decodeBody(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		httpresp.BadRequest(c, "invalid request body")
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, "invalid request body")
		return false
	}
	return true
}
func operationMetadata(c *gin.Context, reason string) (string, bool) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" || len(key) > 128 || strings.ContainsAny(key, "\r\n") {
		httpresp.BadRequest(c, "Idempotency-Key is required (up to 128 characters)")
		return "", false
	}
	if utf8.RuneCountInString(strings.TrimSpace(reason)) < 3 || utf8.RuneCountInString(reason) > 500 {
		httpresp.BadRequest(c, "reason must contain 3 to 500 characters")
		return "", false
	}
	return key, true
}
func requestHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// claimAudit reserves a key atomically. It must be inside the same transaction
// as the business write; a repeated key with a changed payload is rejected.
func claimAudit(tx *gorm.DB, row *AuditEvent) (bool, error) {
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "actor_id"}, {Name: "idempotency_key"}}, DoNothing: true}).Create(row)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	var existing AuditEvent
	if err := tx.Where("actor_id = ? AND idempotency_key = ?", row.ActorID, row.IdempotencyKey).Take(&existing).Error; err != nil {
		return false, err
	}
	if existing.RequestHash != row.RequestHash || existing.Action != row.Action || existing.TargetID != row.TargetID {
		return false, errKeyConflict
	}
	*row = existing
	return false, nil
}

func (m *Module) getSettings(c *gin.Context) {
	settings, err := ReadSettings(c.Request.Context(), m.deps.DB)
	if queryFailed(c, err) {
		return
	}
	httpresp.OK(c, settings)
}

type settingsPatch struct {
	BrandName        *string `json:"brand_name"`
	Description      *string `json:"description"`
	SupportEmail     *string `json:"support_email"`
	SupportURL       *string `json:"support_url"`
	ExportCreditCost *int64  `json:"export_credit_cost"`
	Reason           string  `json:"reason"`
}

func validSettingsPatch(p *settingsPatch) bool {
	if p.BrandName == nil && p.Description == nil && p.SupportEmail == nil && p.SupportURL == nil && p.ExportCreditCost == nil {
		return false
	}
	for value, max := range map[*string]int{p.BrandName: 100, p.Description: 500, p.SupportEmail: 255, p.SupportURL: 1024} {
		if value == nil {
			continue
		}
		*value = strings.TrimSpace(*value)
		if !utf8.ValidString(*value) || utf8.RuneCountInString(*value) > max || strings.ContainsAny(*value, "\x00\r\n") {
			return false
		}
	}
	if p.BrandName != nil && *p.BrandName == "" {
		return false
	}
	if p.SupportEmail != nil && *p.SupportEmail != "" {
		a, err := mail.ParseAddress(*p.SupportEmail)
		if err != nil || a.Address != *p.SupportEmail {
			return false
		}
	}
	if p.SupportURL != nil && *p.SupportURL != "" {
		u, err := url.Parse(*p.SupportURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return false
		}
	}
	return p.ExportCreditCost == nil || (*p.ExportCreditCost >= 1 && *p.ExportCreditCost <= 1000000)
}

func (m *Module) patchSettings(c *gin.Context) {
	actor := userID(c)
	if actor == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	var patch settingsPatch
	if !decodeBody(c, &patch) {
		return
	}
	patch.Reason = strings.TrimSpace(patch.Reason)
	key, ok := operationMetadata(c, patch.Reason)
	if !ok {
		return
	}
	if !validSettingsPatch(&patch) {
		httpresp.BadRequest(c, "invalid or empty settings patch")
		return
	}
	audit := AuditEvent{ActorID: actor, IdempotencyKey: key, Action: "settings.update", TargetID: "site", Reason: patch.Reason, RequestHash: requestHash(patch), Status: "succeeded"}
	var result SiteSettings
	err := db.Transaction(func(tx *gorm.DB) error {
		created, err := claimAudit(tx, &audit)
		if err != nil {
			return err
		}
		if !created {
			return json.Unmarshal([]byte(audit.Details), &result)
		}
		defaults := DefaultSettings()
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&defaults).Error; err != nil {
			return err
		}
		updates := map[string]any{}
		if patch.BrandName != nil {
			updates["brand_name"] = *patch.BrandName
		}
		if patch.Description != nil {
			updates["description"] = *patch.Description
		}
		if patch.SupportEmail != nil {
			updates["support_email"] = *patch.SupportEmail
		}
		if patch.SupportURL != nil {
			updates["support_url"] = *patch.SupportURL
		}
		if patch.ExportCreditCost != nil {
			updates["export_credit_cost"] = *patch.ExportCreditCost
		}
		if err := tx.Model(&SiteSettings{}).Where("id = ?", 1).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", 1).Take(&result).Error; err != nil {
			return err
		}
		raw, _ := json.Marshal(result)
		return tx.Model(&AuditEvent{}).Where("id = ?", audit.ID).Update("details", string(raw)).Error
	})
	if errors.Is(err, errKeyConflict) {
		httpresp.Conflict(c, err.Error())
		return
	}
	if queryFailed(c, err) {
		return
	}
	httpresp.OK(c, result)
}
