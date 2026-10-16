package operations

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/foundation/ossx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const siteLogoPrefix = "/api/v1/site/logo/"

func uploadedLogoID(value string) string {
	if !strings.HasPrefix(value, siteLogoPrefix) {
		return ""
	}
	id, err := uuid.Parse(strings.TrimPrefix(value, siteLogoPrefix))
	if err != nil || siteLogoPrefix+id.String() != value {
		return ""
	}
	return id.String()
}

func validLogoURL(value string) bool {
	if value == "" {
		return true
	}
	if strings.ContainsAny(value, "\\\x00\r\n\t") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}
	if strings.HasPrefix(value, "/api/") {
		return uploadedLogoID(value) != ""
	}
	return (u.Scheme == "https" && u.Hostname() != "") || (strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && u.Host == "" && u.Scheme == "")
}

// Private uploads remain private until explicitly selected as the site's logo.
// Replacing or resetting the logo revokes public access to the previous upload.
func (m *Module) siteLogo(c *gin.Context) {
	db := m.db(c)
	if db == nil {
		return
	}
	settings, err := ReadSettings(c.Request.Context(), db)
	if queryFailed(c, err) {
		return
	}
	id := uploadedLogoID(settings.LogoURL)
	if id == "" || id != c.Param("id") {
		httpresp.NotFound(c, "logo image not found")
		return
	}
	var row Upload
	if err := db.Where("id = ?", id).Take(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		httpresp.NotFound(c, "logo image not found")
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
		httpresp.NotFound(c, "logo image not found")
		return
	}
	if queryFailed(c, err) {
		return
	}
	defer f.Close()
	c.Header("Cache-Control", "public, max-age=0, must-revalidate")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, row.Size, row.ContentType, io.LimitReader(f, row.Size), nil)
}
