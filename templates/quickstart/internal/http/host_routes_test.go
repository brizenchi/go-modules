package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/gin-gonic/gin"
)

func TestHostRoutesDoNotExposeDefaultChatProduct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerHostRoutes(hostapi.Deps{Modules: &platform.Modules{}}, hostapi.Groups{
		Public: router.Group("/api/v1"),
		User:   router.Group("/api/v1"),
		Admin:  router.Group("/api/v1"),
	})
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/ai/capabilities"},
		{http.MethodPost, "/api/v1/ai/chat"},
	} {
		t.Run(route.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(route.method, route.path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", response.Code)
			}
		})
	}
}
