package ginx

import (
	"log/slog"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// Recover catches panics, logs them via slog (with stack), and responds
// 500 with a generic envelope. Adopts foundation/httpresp's shape via a
// JSON literal so this package doesn't have to depend on httpresp.
//
// Use as the FIRST middleware so it sees panics from all subsequent ones.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				var buf [4096]byte
				n := runtime.Stack(buf[:], false)
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"recover", r,
					"stack", string(buf[:n]),
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"code": http.StatusInternalServerError,
						"msg":  "internal server error",
						"data": nil,
					})
				}
			}
		}()
		c.Next()
	}
}
