package note

import (
	"errors"
	"net/http"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/gin-gonic/gin"
)

// Handler translates HTTP to service calls. No business rules here.
type Handler struct {
	svc *service
}

func newHandler(svc *service) *Handler {
	return &Handler{svc: svc}
}

type createRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (h *Handler) Create(c *gin.Context) {
	// Identity is set by the shared RequireUser middleware; on Groups.User
	// it is always non-nil.
	userID := authhttp.Authenticated(c).UserID

	var req createRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 320000)
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "invalid body")
		return
	}

	n, err := h.svc.Create(c.Request.Context(), userID, req.Title, req.Body)
	if err != nil {
		if errors.Is(err, ErrEmptyTitle) || errors.Is(err, ErrInvalidContent) {
			httpresp.BadRequest(c, err.Error())
			return
		}
		httpresp.InternalError(c, "unable to save note")
		return
	}
	httpresp.OK(c, n)
}

func (h *Handler) ListMine(c *gin.Context) {
	userID := authhttp.Authenticated(c).UserID

	rows, err := h.svc.ListMine(c.Request.Context(), userID)
	if err != nil {
		httpresp.InternalError(c, "unable to load notes")
		return
	}
	httpresp.OK(c, gin.H{"list": rows})
}

func (h *Handler) CountAll(c *gin.Context) {
	total, err := h.svc.CountAll(c.Request.Context())
	if err != nil {
		httpresp.InternalError(c, "unable to count notes")
		return
	}
	httpresp.OK(c, gin.H{"total": total})
}
