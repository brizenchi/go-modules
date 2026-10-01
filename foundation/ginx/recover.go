package ginx

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Recover catches panics, logs them via slog (with the full stack), marks
// the active OpenTelemetry span as failed, and responds 500 with a
// generic envelope. Adopts foundation/httpresp's shape via a JSON literal
// so this package doesn't have to depend on httpresp.
//
// Place it after RequestID, tracing.Trace and AccessLog so the recovered
// request still carries its request id and span, ends as a 500 span, and
// produces an access-log record. Panics raised by those outer middleware
// are left to net/http's own recovery.
//
// http.ErrAbortHandler is re-raised so net/http can abort the response
// silently. A panic caused by the client closing the connection is logged
// at WARN and no response is written.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if r == http.ErrAbortHandler { //nolint:errorlint // sentinel compared by identity, as net/http does
				panic(r)
			}
			ctx := c.Request.Context()
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			if brokenPipe(err) {
				slog.WarnContext(ctx, "client connection closed",
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"error", err,
				)
				_ = c.Error(err)
				c.Abort()
				return
			}

			stack := string(debug.Stack())
			slog.ErrorContext(ctx, "panic recovered",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"recover", fmt.Sprint(r),
				"stack", stack,
			)
			span := trace.SpanFromContext(ctx)
			span.RecordError(err, trace.WithAttributes(
				attribute.Bool("exception.escaped", true),
				attribute.String("exception.stacktrace", stack),
			))
			span.SetStatus(codes.Error, "panic recovered")
			_ = c.Error(err)

			if !c.Writer.Written() {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
					"msg":  "internal server error",
					"data": nil,
				})
				return
			}
			c.Abort()
		}()
		c.Next()
	}
}

func brokenPipe(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	var sysErr *os.SyscallError
	if errors.As(opErr, &sysErr) {
		return errors.Is(sysErr.Err, syscall.EPIPE) || errors.Is(sysErr.Err, syscall.ECONNRESET)
	}
	return false
}
