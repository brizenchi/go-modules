// TEMPLATE-OWNED — avoid editing; changes here conflict on upgrade.
package middleware

import (
	"github.com/brizenchi/go-modules/foundation/ginx"
	"github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	ServiceName    string
	AllowedOrigins []string
}

func BuildRouter(cfg RouterConfig, router *apphttp.Router) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000"}
	}
	skip := []string{"/health"}
	// Order matters:
	//   CORS       answers preflights before they become spans and logs.
	//   RequestID  must precede tracing so the id lands on the server span.
	//   Middleware opens the server span; everything below runs inside it.
	//   AccessLog  logs after the handler with request_id/trace_id/span_id.
	//   Recover    is innermost so a panic still ends as a 500 span, a 500
	//              access-log record and a stack trace linked to the trace.
	r.Use(
		ginx.CORS(ginx.CORSConfig{
			AllowedOrigins:   origins,
			AllowCredentials: true,
			AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key", "X-Request-ID", "traceparent", "tracestate", "baggage"},
			ExposedHeaders:   []string{"X-Request-ID"},
		}),
		ginx.RequestID(),
		tracing.Middleware(tracing.MiddlewareConfig{ServiceName: cfg.ServiceName, SkipPaths: skip}),
		ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: skip}),
		ginx.Recover(),
		ginx.NoCache(),
		ginx.Secure(ginx.SecureConfig{}),
	)
	r.GET("/health", apphttp.HealthHandler)

	if router != nil {
		public := r.Group("/api/v1")

		user := r.Group("/api/v1")
		user.Use(router.RequireUser())

		admin := r.Group("/api/v1")
		admin.Use(router.RequireAdmin())

		router.Mount(hostapi.Groups{Public: public, User: user, Admin: admin})
	}

	return r
}
