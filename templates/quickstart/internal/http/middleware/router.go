package middleware

import (
	"github.com/brizenchi/go-modules/foundation/ginx"
	"github.com/brizenchi/go-modules/foundation/tracing"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	ServiceName string
}

func BuildRouter(cfg RouterConfig, stack apphttp.RouteRegistrar) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		ginx.Recover(),
		ginx.RequestID(),
		tracing.Trace(cfg.ServiceName),
		ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: []string{"/health"}}),
	)
	r.Use(ginx.CORS(ginx.CORSConfig{AllowedOrigins: []string{"*"}}))
	r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{}))
	r.GET("/health", apphttp.HealthHandler)

	if stack != nil {
		publicGroup := r.Group("/api/v1")
		userGroup := r.Group("/api/v1")
		userGroup.Use(stack.RequireUser())
		stack.Mount(publicGroup, userGroup)
	}

	return r
}
