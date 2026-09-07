package operations

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (m *Module) getIntegrations(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if m.deps.ServiceSettings == nil {
		httpresp.Custom(c, 503, 503, "integration settings unavailable", nil)
		return
	}
	result, err := m.deps.ServiceSettings.Snapshot(c.Request.Context())
	if integrationError(c, err) {
		return
	}
	httpresp.OK(c, result)
}

// Reasons are operator prose, never a second credential store. Redact common
// provider credentials accidentally pasted into the reason before auditing it.
var credentialInReason = regexp.MustCompile(`\b(?:[sr]k_(?:test|live)_|whsec_|re_)[A-Za-z0-9_\-]+`)

func (m *Module) patchIntegration(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	actor := userID(c)
	if actor == "" {
		return
	}
	if m.deps.ServiceSettings == nil || m.deps.DB == nil {
		httpresp.Custom(c, 503, 503, "integration settings unavailable", nil)
		return
	}
	var patch serviceconfig.Patch
	if !decodeBody(c, &patch) {
		return
	}
	patch.Reason = strings.TrimSpace(patch.Reason)
	key, ok := operationMetadata(c, patch.Reason)
	if !ok {
		return
	}
	provider := c.Param("provider")
	// The one-way digest distinguishes retries without retaining request JSON.
	digest := requestHash(patch)
	reason := patch.Reason
	for _, secret := range patch.Secrets {
		if secret != "" {
			reason = strings.ReplaceAll(reason, secret, "[redacted]")
		}
	}
	reason = credentialInReason.ReplaceAllString(reason, "[redacted]")
	audit := AuditEvent{ActorID: actor, IdempotencyKey: key, Action: "integration.update", TargetID: provider, Reason: reason, RequestHash: digest, Status: "succeeded"}
	var result serviceconfig.Provider
	replayed := false
	// Silence the complete transaction, including its audit reservation, because
	// a deployment logger can otherwise expand credential-bearing SQL parameters.
	err := serviceconfig.QuietDB(m.deps.DB).WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		created, err := claimAudit(tx, &audit)
		if err != nil {
			return err
		}
		if !created {
			replayed = true
			if err := json.Unmarshal([]byte(audit.Details), &result); err != nil {
				return serviceconfig.ErrStorage
			}
			return nil
		}
		result, err = m.deps.ServiceSettings.Update(c.Request.Context(), tx, provider, patch)
		if err != nil {
			return err
		}
		// Provider contains only public field values and configured booleans.
		raw, err := json.Marshal(result)
		if err != nil {
			return serviceconfig.ErrStorage
		}
		return tx.Model(&AuditEvent{}).Where("id = ?", audit.ID).Update("details", string(raw)).Error
	})
	if integrationError(c, err) {
		return
	}
	if replayed {
		// Audit details describe the original save. A retry after a restart or a
		// newer edit must report today's active state and optimistic version.
		snapshot, err := m.deps.ServiceSettings.Snapshot(c.Request.Context())
		if integrationError(c, err) {
			return
		}
		for _, current := range snapshot.Providers {
			if current.Provider == provider {
				result = current
				break
			}
		}
	}
	httpresp.OK(c, result)
}

func integrationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var validation *serviceconfig.ValidationError
	switch {
	case errors.Is(err, serviceconfig.ErrVersion), errors.Is(err, errKeyConflict):
		httpresp.Conflict(c, err.Error())
	case errors.Is(err, serviceconfig.ErrProvider):
		httpresp.NotFound(c, "unsupported integration")
	case errors.Is(err, serviceconfig.ErrSchema):
		httpresp.Custom(c, 503, 503, serviceconfig.ErrSchema.Error(), nil)
	case errors.As(err, &validation):
		httpresp.BadRequest(c, validation.Message)
	default:
		// Raw database and SQL errors must never cross this boundary.
		httpresp.Custom(c, 503, 503, "integration settings unavailable", nil)
	}
	return true
}
