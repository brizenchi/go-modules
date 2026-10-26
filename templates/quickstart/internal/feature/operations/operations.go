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
	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/brizenchi/quickstart-template/internal/checkoutlog"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
	"github.com/brizenchi/quickstart-template/internal/feature/writer"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	deps        hostapi.Deps
	storageOnce sync.Once
	bucket      ossx.Bucket
	storageErr  error
	writerOnce  sync.Once
	writer      *writer.Service
}

func New(deps hostapi.Deps) *Module { return &Module{deps: deps.WithServices()} }

func (m *Module) Register(g hostapi.Groups) {
	if g.Public != nil {
		g.Public.GET("/site/settings", m.getSettings)
		g.Public.GET("/site/logo/:id", m.siteLogo)
		g.Public.GET("/site/images/:id", m.siteImage)
		g.Public.GET("/site/articles", m.publicArticles)
		g.Public.GET("/site/articles/resolve", m.resolveArticle)
	}
	if g.User != nil {
		g.User.GET("/billing/purchases", m.purchases)
		g.User.GET("/billing/credit-checkouts/:id", m.creditCheckoutResult)
		g.User.POST("/billing/credit-checkouts/:id/confirmation", m.confirmCreditCheckout)
		g.User.POST("/uploads/images", m.uploadImage)
		g.User.GET("/uploads/images", m.listImages)
		g.User.GET("/uploads/images/:id", m.getImage)
	}
	if g.Admin == nil {
		return
	}
	admin := g.Admin.Group("/admin", requireAdmin, m.auditFailures)
	admin.GET("/access", func(c *gin.Context) { httpresp.OK(c, gin.H{"authorized": true}) })
	admin.GET("/overview", m.overview)
	admin.GET("/users", m.users)
	admin.GET("/users/:id/access", m.userAccess)
	admin.PATCH("/users/:id/membership", m.patchMembership)
	admin.POST("/users/:id/credit-deductions", m.deductCredits)
	admin.GET("/subscriptions", m.subscriptions)
	admin.GET("/orders", m.orders)
	admin.GET("/referrals", m.referrals)
	admin.POST("/referrals/:id/retry-reward", m.retryReward)
	admin.GET("/articles", m.articles)
	admin.GET("/articles/:id", m.article)
	admin.POST("/articles", m.saveArticle)
	admin.POST("/articles/import", m.importArticles)
	admin.POST("/articles/drafts", m.draftArticle)
	admin.POST("/articles/translations", m.translateArticle)
	admin.POST("/site-images", m.uploadSiteImage)
	admin.PUT("/articles/:id", m.saveArticle)
	admin.POST("/articles/:id/publications", m.publishArticle)
	admin.GET("/audit", m.audit)
	admin.GET("/settings", m.getSettings)
	admin.PATCH("/settings", m.patchSettings)
	admin.GET("/integrations", m.getIntegrations)
	admin.PATCH("/integrations/:provider", func(c *gin.Context) {
		httpresp.Custom(c, 403, 403, "credentials are deployment-managed", gin.H{"reason": "INTEGRATION_DEPLOYMENT_MANAGED"})
	})
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
	ID               uint   `gorm:"primaryKey" json:"-"`
	BrandName        string `gorm:"type:varchar(100);not null" json:"brand_name"`
	LogoURL          string `gorm:"type:varchar(1024);not null;default:''" json:"logo_url"`
	Description      string `gorm:"type:varchar(500);not null" json:"description"`
	DescriptionZH    string `gorm:"type:varchar(500);not null;default:''" json:"description_zh"`
	HeadlineEN       string `gorm:"type:varchar(160);not null;default:''" json:"headline_en"`
	HeadlineZH       string `gorm:"type:varchar(160);not null;default:''" json:"headline_zh"`
	SenderEmail      string `gorm:"type:varchar(255);not null;default:''" json:"sender_email"`
	SenderName       string `gorm:"type:varchar(100);not null;default:''" json:"sender_name"`
	SupportEmail     string `gorm:"type:varchar(255);not null" json:"support_email"`
	SupportURL       string `gorm:"type:varchar(1024);not null" json:"support_url"`
	ExportCreditCost int64  `gorm:"not null" json:"export_credit_cost"`
	// Navigation is nil until an administrator saves menus; the frontend then uses its defaults.
	Navigation *Navigation `gorm:"type:text;serializer:json" json:"navigation"`
	UpdatedAt  time.Time   `json:"-"`
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

type AuditEvent = audit.Operation

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

func Models() []any {
	models := append([]any{&SiteSettings{}, &Upload{}, &SiteImage{}, &checkoutlog.Record{}}, audit.Models()...)
	models = append(models, content.Models()...)
	return append(models, serviceconfig.Models()...)
}
