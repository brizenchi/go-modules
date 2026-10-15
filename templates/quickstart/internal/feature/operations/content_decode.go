package operations

import (
	"encoding/json"
	"errors"
	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/gin-gonic/gin"
	"io"
)

func decodeContentBody(c *gin.Context, target any) bool {
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		httpresp.BadRequest(c, "invalid content body")
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, "invalid content body")
		return false
	}
	return true
}
