// Package operations owns the host's operator console, public business settings,
// and private image uploads. It never exposes provider credentials or webhook payloads.
package operations

import (
	"context"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/foundation/ossx"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	deps        hostapi.Deps
	storageOnce sync.Once
	bucket      ossx.Bucket
	storageErr  error
}

func New(deps hostapi.Deps) *Module { return &Module{deps: deps} }

func (m *Module) Register(g hostapi.Groups) {
	if g.Public != nil {
		g.Public.GET("/site/settings", m.getSettings)
	}
	if g.User != nil {
		g.User.POST("/uploads/images", m.uploadImage)
		g.User.GET("/uploads/images", m.listImages)
		g.User.GET("/uploads/images/:id", m.getImage)
	}
	if g.Admin == nil {
		return
	}
	admin := g.Admin.Group("/admin", requireAdmin)
	admin.GET("/overview", m.overview)
	admin.GET("/users", m.users)
	admin.GET("/subscriptions", m.subscriptions)
	admin.GET("/orders", m.orders)
	admin.GET("/referrals", m.referrals)
	admin.POST("/referrals/:id/retry-reward", m.retryReward)
	admin.GET("/audit", m.audit)
	admin.GET("/settings", m.getSettings)
	admin.PATCH("/settings", m.patchSettings)
}

func requireAdmin(c *gin.Context) {
	id := authhttp.Authenticated(c)
	if id == nil || id.UserID == "" {
		httpresp.Unauthorized(c, "unauthorized")
		return
	}
	if id.Role != authdomain.RoleAdmin {
		httpresp.Forbidden(c, "admin role required")
		return
	}
	c.Next()
}

func userID(c *gin.Context) string {
	id := authhttp.Authenticated(c)
	if id == nil || id.UserID == "" {
		httpresp.Unauthorized(c, "unauthorized")
		return ""
	}
	return id.UserID
}

type SiteSettings struct {
	ID               uint      `gorm:"primaryKey" json:"-"`
	BrandName        string    `gorm:"type:varchar(100);not null" json:"brand_name"`
	Description      string    `gorm:"type:varchar(500);not null" json:"description"`
	SupportEmail     string    `gorm:"type:varchar(255);not null" json:"support_email"`
	SupportURL       string    `gorm:"type:varchar(1024);not null" json:"support_url"`
	ExportCreditCost int64     `gorm:"not null" json:"export_credit_cost"`
	UpdatedAt        time.Time `json:"-"`
}

func (SiteSettings) TableName() string { return "site_settings" }

func DefaultSettings() SiteSettings {
	return SiteSettings{ID: 1, BrandName: "SaaS Starter", Description: "Launch your SaaS with authentication, billing and referrals.", ExportCreditCost: 1}
}

func ReadSettings(ctx context.Context, db *gorm.DB) (*SiteSettings, error) {
	settings := DefaultSettings()
	if db == nil {
		return &settings, nil
	}
	err := db.WithContext(ctx).Where("id = ?", 1).Take(&settings).Error
	if err == gorm.ErrRecordNotFound {
		return &settings, nil
	}
	return &settings, err
}

type AuditEvent struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID        string    `gorm:"type:varchar(36);not null;index;uniqueIndex:uniq_operations_actor_key" json:"actor_id"`
	IdempotencyKey string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_operations_actor_key" json:"idempotency_key"`
	Action         string    `gorm:"type:varchar(64);not null;index" json:"action"`
	TargetID       string    `gorm:"type:varchar(255);not null" json:"target_id"`
	Reason         string    `gorm:"type:varchar(500);not null" json:"reason"`
	RequestHash    string    `gorm:"type:varchar(64);not null" json:"-"`
	Status         string    `gorm:"type:varchar(32);not null" json:"status"`
	Details        string    `gorm:"type:text" json:"details"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AuditEvent) TableName() string { return "operations_audit_events" }

type Upload struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID      string    `gorm:"type:varchar(36);not null;index" json:"-"`
	StorageKey  string    `gorm:"type:varchar(255);not null" json:"-"`
	Provider    string    `gorm:"type:varchar(16);not null" json:"-"`
	ContentType string    `gorm:"type:varchar(32);not null" json:"content_type"`
	Size        int64     `gorm:"not null" json:"size"`
	Filename    string    `gorm:"type:varchar(100);not null" json:"filename"`
	URL         string    `gorm:"-" json:"url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Upload) TableName() string { return "private_image_uploads" }

func Models() []any { return []any{&SiteSettings{}, &AuditEvent{}, &Upload{}} }
