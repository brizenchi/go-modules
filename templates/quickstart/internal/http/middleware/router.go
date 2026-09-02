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
	r.Use(
		ginx.Recover(),
		ginx.RequestID(),
		tracing.Trace(cfg.ServiceName),
		ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: []string{"/health"}}),
	)
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000"}
	}
	r.Use(ginx.CORS(ginx.CORSConfig{AllowedOrigins: origins}))
	r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{}))
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
