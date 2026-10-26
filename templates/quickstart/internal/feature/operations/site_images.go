package operations

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF dimensions for uploaded article images.
	_ "image/jpeg" // Register JPEG dimensions for uploaded article images.
	_ "image/png"  // Register PNG dimensions for uploaded article images.
	"io"
	"net/http"
	"os"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/foundation/ossx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SiteImage is an administrator-uploaded image for published articles. Unlike
// private uploads it is public by URL, so articles and search engines can load it.
type SiteImage struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedBy   string    `gorm:"type:varchar(36);not null;index" json:"-"`
	StorageKey  string    `gorm:"type:varchar(255);not null" json:"-"`
	Provider    string    `gorm:"type:varchar(16);not null" json:"-"`
	ContentType string    `gorm:"type:varchar(32);not null" json:"content_type"`
	Size        int64     `gorm:"not null" json:"size"`
	Width       int       `gorm:"not null;default:0" json:"width"`
	Height      int       `gorm:"not null;default:0" json:"height"`
	URL         string    `gorm:"-" json:"url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SiteImage) TableName() string { return "site_images" }

const siteImagePath = "/api/v1/site/images/"

// siteImageURL carries known dimensions so pages can reserve space before loading.
func siteImageURL(row SiteImage) string {
	if row.Width > 0 && row.Height > 0 {
		return fmt.Sprintf("%s%s?w=%d&h=%d", siteImagePath, row.ID, row.Width, row.Height)
	}
	return siteImagePath + row.ID
}

func (m *Module) uploadSiteImage(c *gin.Context) {
	actor := userID(c)
	if actor == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	body, kind, ext, ok := m.readUploadedImage(c)
	if !ok {
		return
	}
	row := SiteImage{ID: uuid.NewString(), CreatedBy: actor, Provider: m.storageProvider(), ContentType: kind, Size: int64(len(body))}
	// WebP dimensions are not decoded; such images render without reserved space.
	if config, _, err := image.DecodeConfig(bytes.NewReader(body)); err == nil {
		row.Width, row.Height = config.Width, config.Height
	}
	row.StorageKey = "site-images/" + row.ID + ext
	if err := m.putImage(c.Request.Context(), row.StorageKey, body, kind); err != nil {
		httpresp.InternalError(c, "image could not be stored")
		return
	}
	if err := db.Create(&row).Error; err != nil {
		m.removeImage(c.Request.Context(), row.StorageKey)
		httpresp.InternalError(c, "image could not be saved")
		return
	}
	row.URL = siteImageURL(row)
	httpresp.OK(c, row)
}

func (m *Module) siteImage(c *gin.Context) {
	db := m.db(c)
	if db == nil {
		return
	}
	var row SiteImage
	if err := db.WithContext(c.Request.Context()).Where("id = ?", c.Param("id")).Take(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		httpresp.NotFound(c, "image not found")
		return
	} else if queryFailed(c, err) {
		return
	}
	if err := m.prepareStorage(); err != nil || row.Provider != m.storageProvider() {
		httpresp.Custom(c, 503, 503, "image storage is not configured", nil)
		return
	}
	f, err := m.readImage(c.Request.Context(), row.StorageKey)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ossx.ErrNotFound) {
		httpresp.NotFound(c, "image not found")
		return
	}
	if queryFailed(c, err) {
		return
	}
	defer f.Close()
	// The id never points at different bytes, so browsers and CDNs may keep it;
	// this replaces the API-wide no-cache headers.
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Writer.Header().Del("Expires")
	c.Header("Last-Modified", row.CreatedAt.UTC().Format(http.TimeFormat))
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, row.Size, row.ContentType, io.LimitReader(f, row.Size), nil)
}
