// Package httpresp provides a uniform JSON envelope for HTTP responses.
//
// Envelope shape (matches the convention used by many backends):
//
//	{ "code": <int>, "msg": "<string>", "data": <any|null> }
//
// Helpers always set the HTTP status code AND the envelope code; for
// most errors they're equal. The exception is "soft errors" (logical
// failures the client should display rather than retry): callers can
// use OKWith(code, msg, data) to return HTTP 200 with a non-success
// code in the body — common pattern for form validation errors.
//
// Helpers call c.Abort() on every error response so middleware after
// the handler sees the chain as terminated.
package httpresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the shape returned by every helper.
type Envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data"`
}

// Standard codes embedded in the envelope.
const (
	CodeOK           = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeConflict     = 409
	CodeTooManyReq   = 429
	CodeInternal     = 500
)

const msgOK = "ok"

// OK responds 200 with {code:200, msg:"ok", data}.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: CodeOK, Msg: msgOK, Data: data})
}

// OKWith responds 200 with the supplied code/msg in the envelope.
// Use for "soft" errors (validation messages etc.) the client should
// display rather than treat as a transport-level failure.
func OKWith(c *gin.Context, code int, msg string, data any) {
	c.JSON(http.StatusOK, Envelope{Code: code, Msg: msg, Data: data})
}

// BadRequest responds 400.
func BadRequest(c *gin.Context, msg string) { abortJSON(c, http.StatusBadRequest, CodeBadRequest, msg) }

// Unauthorized responds 401.
func Unauthorized(c *gin.Context, msg string) {
	abortJSON(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

// Forbidden responds 403.
func Forbidden(c *gin.Context, msg string) { abortJSON(c, http.StatusForbidden, CodeForbidden, msg) }

// NotFound responds 404.
func NotFound(c *gin.Context, msg string) { abortJSON(c, http.StatusNotFound, CodeNotFound, msg) }

// Conflict responds 409.
func Conflict(c *gin.Context, msg string) { abortJSON(c, http.StatusConflict, CodeConflict, msg) }

// TooManyRequests responds 429.
func TooManyRequests(c *gin.Context, msg string) {
	abortJSON(c, http.StatusTooManyRequests, CodeTooManyReq, msg)
}

// InternalError responds 500.
func InternalError(c *gin.Context, msg string) {
	abortJSON(c, http.StatusInternalServerError, CodeInternal, msg)
}

// Custom responds with the given HTTP status + envelope code/msg + data.
//
// Use sparingly; the named helpers cover the common cases.
func Custom(c *gin.Context, httpStatus, code int, msg string, data any) {
	c.AbortWithStatusJSON(httpStatus, Envelope{Code: code, Msg: msg, Data: data})
}

func abortJSON(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Envelope{Code: code, Msg: msg})
}
