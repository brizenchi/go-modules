package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errKeyConflict = audit.ErrConflict

func decodeBody(c *gin.Context, target any) bool { return decodeBodyLimit(c, target, 8192) }

// Settings carry navigation menus, so their body may exceed the default limit.
const settingsBodyLimit = 32768

func decodeBodyLimit(c *gin.Context, target any, limit int64) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
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
func requestHash(value any) string { return audit.Hash(value) }
func claimAudit(tx *gorm.DB, row *AuditEvent) (bool, error) {
	return audit.New().Claim(tx.Statement.Context, tx, row)
}

func (m *Module) getSettings(c *gin.Context) {
	settings, err := ReadSettings(c.Request.Context(), m.deps.DB)
	if queryFailed(c, err) {
		return
	}
	httpresp.OK(c, gin.H{"brand_name": settings.BrandName, "logo_url": settings.LogoURL, "description": settings.Description, "description_zh": settings.DescriptionZH, "headline_en": settings.HeadlineEN, "headline_zh": settings.HeadlineZH, "sender_email": settings.SenderEmail, "sender_name": settings.SenderName, "support_email": settings.SupportEmail, "support_url": settings.SupportURL, "export_credit_cost": settings.ExportCreditCost, "navigation": settings.Navigation, "presentation_version": 1})
}

type settingsPatch struct {
	BrandName        *string     `json:"brand_name"`
	LogoURL          *string     `json:"logo_url"`
	Description      *string     `json:"description"`
	SupportEmail     *string     `json:"support_email"`
	SupportURL       *string     `json:"support_url"`
	ExportCreditCost *int64      `json:"export_credit_cost"`
	DescriptionZH    *string     `json:"description_zh"`
	HeadlineEN       *string     `json:"headline_en"`
	HeadlineZH       *string     `json:"headline_zh"`
	SenderEmail      *string     `json:"sender_email"`
	SenderName       *string     `json:"sender_name"`
	Navigation       *Navigation `json:"navigation"`
	Reason           string      `json:"reason"`
}

func validSettingsPatch(p *settingsPatch) bool {
	if p.LogoURL == nil && p.BrandName == nil && p.Description == nil && p.SupportEmail == nil && p.SupportURL == nil && p.ExportCreditCost == nil && p.DescriptionZH == nil && p.HeadlineEN == nil && p.HeadlineZH == nil && p.SenderEmail == nil && p.SenderName == nil && p.Navigation == nil {
		return false
	}
	if p.Navigation != nil && !validNavigation(p.Navigation) {
		return false
	}
	for value, max := range map[*string]int{p.LogoURL: 1024, p.BrandName: 100, p.Description: 500, p.SupportEmail: 255, p.SupportURL: 1024, p.DescriptionZH: 500, p.HeadlineEN: 160, p.HeadlineZH: 160, p.SenderEmail: 255, p.SenderName: 100} {
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
	if p.SenderEmail != nil && *p.SenderEmail != "" {
		a, err := mail.ParseAddress(*p.SenderEmail)
		if err != nil || a.Address != *p.SenderEmail {
			return false
		}
	}
	if p.SupportURL != nil && *p.SupportURL != "" {
		u, err := url.Parse(*p.SupportURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return false
		}
	}
	if p.LogoURL != nil && !validLogoURL(*p.LogoURL) {
		return false
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
	if !decodeBodyLimit(c, &patch, settingsBodyLimit) {
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
			var change struct {
				After *SiteSettings `json:"after"`
			}
			if err := json.Unmarshal([]byte(audit.Details), &change); err != nil {
				return err
			}
			if change.After != nil {
				result = *change.After
				return nil
			}
			return json.Unmarshal([]byte(audit.Details), &result) // Legacy audit snapshots.
		}
		defaults := DefaultSettings()
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&defaults).Error; err != nil {
			return err
		}
		var before SiteSettings
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", 1).Take(&before).Error; err != nil {
			return err
		}
		updates := map[string]any{}
		if patch.LogoURL != nil {
			if id := uploadedLogoID(*patch.LogoURL); id != "" && *patch.LogoURL != before.LogoURL {
				// Only the administrator's own private upload can be made public.
				var upload Upload
				if err := tx.Where("id = ? AND user_id = ?", id, actor).Take(&upload).Error; err != nil {
					return err
				}
			}
			updates["logo_url"] = *patch.LogoURL
		}
		for value, column := range map[*string]string{patch.DescriptionZH: "description_zh", patch.HeadlineEN: "headline_en", patch.HeadlineZH: "headline_zh", patch.SenderEmail: "sender_email", patch.SenderName: "sender_name"} {
			if value != nil {
				updates[column] = *value
			}
		}
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
		if patch.Navigation != nil {
			raw, err := json.Marshal(patch.Navigation)
			if err != nil {
				return fmt.Errorf("encode navigation: %w", err)
			}
			updates["navigation"] = string(raw)
		}
		if err := tx.Model(&SiteSettings{}).Where("id = ?", 1).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", 1).Take(&result).Error; err != nil {
			return err
		}
		raw, _ := json.Marshal(struct {
			Before SiteSettings `json:"before"`
			After  SiteSettings `json:"after"`
		}{Before: before, After: result})
		return m.deps.Audit.Finish(c.Request.Context(), tx, &audit, "succeeded", string(raw))
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpresp.NotFound(c, "logo image not found")
		return
	}
	if errors.Is(err, errKeyConflict) {
		httpresp.Conflict(c, err.Error())
		return
	}
	if queryFailed(c, err) {
		return
	}
	httpresp.OK(c, result)
}
