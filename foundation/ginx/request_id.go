package ginx

import (
	"context"

	flog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the canonical header name for the request id.
const HeaderRequestID = "X-Request-ID"

// MaxRequestIDLength bounds an inbound X-Request-ID. Longer values are
// replaced with a generated id.
const MaxRequestIDLength = 128

// ContextKey is the key under which the request id is stored.
type ContextKey string

const RequestIDKey ContextKey = "request_id"

// RequestID assigns/propagates a request id.
//
// Behavior:
//  1. Read X-Request-ID from incoming request.
//  2. If absent or invalid, generate a UUIDv4. A valid id is at most
//     MaxRequestIDLength characters of [A-Za-z0-9._:-], which keeps
//     untrusted input from injecting control characters into logs.
//  3. Store under c.Get("request_id") AND in c.Request.Context() under
//     both RequestIDKey and foundation/slog.RequestIDKey.
//  4. Echo back as X-Request-ID response header.
//
// foundation/slog attaches the id to every *Context log record, and
// foundation/tracing copies it onto the server span.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if !validRequestID(rid) {
			rid = uuid.NewString()
		}
		c.Set(string(RequestIDKey), rid)
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, rid)
		c.Request = c.Request.WithContext(flog.ContextWithRequestID(ctx, rid))
		c.Header(HeaderRequestID, rid)
		c.Next()
	}
}

// RequestIDFromContext returns the request id stored by RequestID(), or "".
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if rid := c.GetString(string(RequestIDKey)); rid != "" {
		return rid
	}
	if c.Request != nil {
		if rid, _ := c.Request.Context().Value(RequestIDKey).(string); rid != "" {
			return rid
		}
		return flog.RequestIDFromContext(c.Request.Context())
	}
	return ""
}

func validRequestID(id string) bool {
	if id == "" || len(id) > MaxRequestIDLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		switch b := id[i]; {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		case b == '-', b == '_', b == '.', b == ':':
		default:
			return false
		}
	}
	return true
}
