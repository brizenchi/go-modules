package ginx

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the canonical header name for the request id.
const HeaderRequestID = "X-Request-ID"

// ContextKey is the key under which the request id is stored.
type ContextKey string

const RequestIDKey ContextKey = "request_id"

// RequestID assigns/propagates a request id.
//
// Behavior:
//  1. Read X-Request-ID from incoming request.
//  2. If absent, generate a UUIDv4.
//  3. Store under c.Get("request_id") AND in c.Request.Context().
//  4. Echo back as X-Request-ID response header.
//
// foundation/slog.With(c) reads RequestIDKey to attach to log records.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(string(RequestIDKey), rid)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), RequestIDKey, rid))
		c.Header(HeaderRequestID, rid)
		c.Next()
	}
}
