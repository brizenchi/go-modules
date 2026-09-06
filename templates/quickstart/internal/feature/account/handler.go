package account

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxProfileRequestBytes int64 = 8 << 10

type Handler struct {
	service *service
}

func newHandler(service *service) *Handler {
	return &Handler{service: service}
}

type updateProfileRequest struct {
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

type profileResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Username      string    `json:"username"`
	AvatarURL     string    `json:"avatar_url"`
	Role          string    `json:"role"`
	Credits       int64     `json:"credits"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (h *Handler) GetProfile(c *gin.Context) {
	identity, ok := authenticatedIdentity(c)
	if !ok {
		return
	}
	account, err := h.service.Get(c.Request.Context(), identity.UserID)
	if err != nil {
		respondProfileError(c, err)
		return
	}
	httpresp.OK(c, newProfileResponse(account, identity.Role))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	identity, ok := authenticatedIdentity(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProfileRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request updateProfileRequest
	if err := decoder.Decode(&request); err != nil {
		httpresp.BadRequest(c, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, "invalid request body")
		return
	}

	account, err := h.service.Update(c.Request.Context(), identity.UserID, profilePatch{
		Username:  request.Username,
		AvatarURL: request.AvatarURL,
	})
	if err != nil {
		respondProfileError(c, err)
		return
	}
	httpresp.OK(c, newProfileResponse(account, identity.Role))
}

func authenticatedIdentity(c *gin.Context) (*authdomain.Identity, bool) {
	identity := authhttp.Authenticated(c)
	if identity == nil || strings.TrimSpace(identity.UserID) == "" {
		httpresp.Unauthorized(c, "unauthorized")
		return nil, false
	}
	return identity, true
}

func newProfileResponse(account *user.User, effectiveRole authdomain.Role) profileResponse {
	role := strings.TrimSpace(string(effectiveRole))
	if role == "" {
		role = account.Role
	}
	return profileResponse{
		ID:            account.ID,
		Email:         account.Email,
		EmailVerified: account.EmailVerified,
		Username:      account.Username,
		AvatarURL:     account.AvatarURL,
		Role:          role,
		Credits:       account.Credits,
		CreatedAt:     account.CreatedAt,
		UpdatedAt:     account.UpdatedAt,
	}
}

func respondProfileError(c *gin.Context, err error) {
	switch {
	case isValidationError(err):
		httpresp.BadRequest(c, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpresp.NotFound(c, "account not found")
	default:
		httpresp.InternalError(c, "account operation failed")
	}
}
