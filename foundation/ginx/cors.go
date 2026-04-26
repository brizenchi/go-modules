// Package ginx is a small collection of standard Gin middleware that
// every service in the org should use.
//
// Each middleware is independent — pick what you need at boot:
//
//	r := gin.New()
//	r.Use(ginx.Recover(), ginx.RequestID(), ginx.AccessLog(), ginx.NoCache(), ginx.Secure())
//	r.Use(ginx.CORS(ginx.CORSConfig{
//	    AllowedOrigins: []string{"https://app.example.com"},
//	}))
package ginx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is the explicit origin allowlist. Use ["*"] for any.
	// When empty, defaults to ["*"] (development-friendly; tighten in prod).
	AllowedOrigins []string

	// AllowedMethods. Empty → all common methods.
	AllowedMethods []string

	// AllowedHeaders. Empty → a sensible default that covers most APIs.
	AllowedHeaders []string

	// ExposedHeaders are sent in Access-Control-Expose-Headers.
	ExposedHeaders []string

	// AllowCredentials lets browsers send cookies/Authorization. Note:
	// when true the wildcard "*" origin must NOT be used (CORS spec).
	AllowCredentials bool

	// MaxAgeSeconds caches the preflight result. Default 86400 (24h).
	MaxAgeSeconds int
}

// CORS returns a CORS middleware configured by cfg.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowedMethods := joinOrDefault(cfg.AllowedMethods,
		"GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
	allowedHeaders := joinOrDefault(cfg.AllowedHeaders,
		"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, "+
			"Authorization, X-Requested-With, Accept, User-Agent, Cache-Control, "+
			"Pragma, Date, If-Modified-Since, X-API-Key, X-Auth-Token, X-Request-ID, "+
			"X-Trace-ID, X-Span-ID, X-Forwarded-For, X-Real-IP, Stripe-Signature")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := "86400"
	if cfg.MaxAgeSeconds > 0 {
		maxAge = itoa(cfg.MaxAgeSeconds)
	}
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && originAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" && !cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", allowedMethods)
		c.Header("Access-Control-Allow-Headers", allowedHeaders)
		if exposed != "" {
			c.Header("Access-Control-Expose-Headers", exposed)
		}
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Max-Age", maxAge)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

func joinOrDefault(parts []string, def string) string {
	if len(parts) == 0 {
		return def
	}
	return strings.Join(parts, ", ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
