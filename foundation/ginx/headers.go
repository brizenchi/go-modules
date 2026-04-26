package ginx

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NoCache writes anti-caching headers; mount on routes that return
// dynamic JSON to prevent shared-cache pollution.
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
		c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Next()
	}
}

// SecureConfig tweaks the Secure middleware. Empty values keep defaults.
type SecureConfig struct {
	// ContentSecurityPolicy. Empty disables.
	ContentSecurityPolicy string
	// HSTS controls Strict-Transport-Security on TLS connections.
	// Empty defaults to "max-age=31536000; includeSubDomains; preload".
	// Set to "off" to disable entirely.
	HSTS string
}

// Secure writes a baseline of security headers.
//
// Defaults:
//   - X-Frame-Options: DENY
//   - X-Content-Type-Options: nosniff
//   - X-XSS-Protection: 1; mode=block
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Strict-Transport-Security (TLS only): 1y + subdomains + preload
//
// Pass an empty SecureConfig{} for sane defaults.
func Secure(cfg SecureConfig) gin.HandlerFunc {
	hsts := cfg.HSTS
	if hsts == "" {
		hsts = "max-age=31536000; includeSubDomains; preload"
	}
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if cfg.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", cfg.ContentSecurityPolicy)
		}
		if c.Request.TLS != nil && hsts != "off" {
			c.Header("Strict-Transport-Security", hsts)
		}
		c.Next()
	}
}
