package httpresp_test

import (
	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/gin-gonic/gin"
)

// Example_oK is the happy-path helper: HTTP 200 + envelope code 200.
func Example_oK() {
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		httpresp.OK(c, gin.H{"id": 42, "email": "alice@example.com"})
	})
}

// Example_softError demonstrates returning HTTP 200 with a non-success
// envelope code — the convention for "validation failed, show this
// message to the user" without triggering retry / error UI.
func Example_softError() {
	r := gin.New()
	r.POST("/signup", func(c *gin.Context) {
		// ... validate ...
		httpresp.OKWith(c, 4001, "email already in use", nil)
	})
}

// Example_hardError aborts the chain with HTTP 401.
func Example_hardError() {
	r := gin.New()
	r.GET("/admin", func(c *gin.Context) {
		httpresp.Unauthorized(c, "missing token")
		// any middleware after this sees c.IsAborted() == true
	})
}
