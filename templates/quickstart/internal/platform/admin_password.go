package platform

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/modules/auth"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminPasswordMinBytes = 12
	adminPasswordMaxBytes = 72 // bcrypt's input limit; never silently truncate.
	adminPasswordCost     = 12
	adminLoginWindow      = time.Minute
	adminLoginIPLimit     = 5
	adminLoginGlobalLimit = 20
)

var errInvalidAdminCredentials = errors.New("invalid admin email or password")

// ValidateAdminPassword runs before the application's database is opened.
// A partial configuration is an error rather than an accidental login fallback.
func (c Config) ValidateAdminPassword() error {
	email := strings.ToLower(strings.TrimSpace(c.Auth.AdminEmail))
	password := c.Auth.AdminPassword
	if c.Auth.AdminEmail == "" && password == "" {
		return nil
	}
	if email == "" || password == "" {
		return fmt.Errorf("auth.admin_email and auth.admin_password must be configured together")
	}
	if !c.AuthEnabled() {
		return fmt.Errorf("admin password login requires auth.enabled=true")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || address.Name != "" || len(email) > 254 {
		return fmt.Errorf("auth.admin_email must be a valid email address without a display name")
	}
	if len(password) < adminPasswordMinBytes || len(password) > adminPasswordMaxBytes || strings.TrimSpace(password) == "" {
		return fmt.Errorf("auth.admin_password must contain 12 to 72 bytes and must not be blank")
	}
	return nil
}

type adminPasswordVerifier struct {
	emailHash [sha256.Size]byte
	hash      []byte
}

// Verify implements the existing credential-verifier port for a private login
// service. The password is never sent to an OTP provider or retained as a code.
func (v *adminPasswordVerifier) Verify(_ context.Context, email, password string) error {
	emailHash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	emailMatches := subtle.ConstantTimeCompare(emailHash[:], v.emailHash[:])
	validLength := len(password) >= adminPasswordMinBytes && len(password) <= adminPasswordMaxBytes
	if !validLength {
		// Still perform one bcrypt check for malformed credentials, without
		// letting bcrypt accept an overlong password's first 72 bytes.
		password = "invalid-password-length"
	}
	passwordErr := bcrypt.CompareHashAndPassword(v.hash, []byte(password))
	if emailMatches != 1 || passwordErr != nil || !validLength {
		return errInvalidAdminCredentials
	}
	return nil
}

type adminPasswordLogin struct {
	login   *authapp.LoginService
	limiter adminLoginLimiter
}

func newAdminPasswordLogin(cfg Config, module *auth.Module) (*adminPasswordLogin, error) {
	if cfg.Auth.AdminPassword == "" || module == nil {
		return nil, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Auth.AdminPassword), adminPasswordCost)
	if err != nil {
		return nil, fmt.Errorf("platform: initialize admin password authentication: %w", err)
	}
	verifier := &adminPasswordVerifier{
		emailHash: sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(cfg.Auth.AdminEmail)))),
		hash:      hash,
	}
	return &adminPasswordLogin{login: authapp.NewLoginService(authapp.LoginDeps{
		Verifier: verifier,
		Users:    module.Deps.UserStore,
		Roles:    module.Deps.RoleResolver,
		Signer:   module.Deps.TokenSigner,
		Bus:      module.Deps.Bus,
		TokenTTL: time.Duration(cfg.Auth.UserJWTExpireHours) * time.Hour,
	})}, nil
}

func (m *Modules) adminPasswordLogin(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if m.adminPassword == nil || m.Auth == nil {
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "admin password login is not configured", nil)
		return
	}
	if retryAfter := m.adminPassword.limiter.reserve(time.Now(), adminLoginPeer(c.Request.RemoteAddr)); retryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int((retryAfter+time.Second-1)/time.Second)))
		httpresp.TooManyRequests(c, "too many admin login attempts")
		return
	}
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !authhttp.BindJSONBody(c, &request, false) {
		return
	}
	// Reuse user creation, role resolution, session issuance, login timestamps
	// and signup/login hooks. This endpoint does not accept referral attribution.
	result, err := m.adminPassword.login.VerifyCode(c.Request.Context(), request.Email, request.Password)
	request.Password = ""
	if errors.Is(err, errInvalidAdminCredentials) {
		httpresp.Unauthorized(c, errInvalidAdminCredentials.Error())
		return
	}
	if err != nil {
		respondAuthError(c, err)
		return
	}
	httpresp.OK(c, verifyResultJSON(result))
}

// Rate limits count every attempt, including successful and malformed requests.
// One configured account has a shared limit, so rotating email addresses or IPs
// cannot bypass it. The map contains at most adminLoginGlobalLimit peer entries.
// This in-memory limit applies per API process; deployments with multiple
// replicas should also apply a shared limit at their ingress.
type adminLoginLimiter struct {
	mu      sync.Mutex
	started time.Time
	total   int
	peers   map[string]int
}

func (l *adminLoginLimiter) reserve(now time.Time, peer string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started.IsZero() || !now.Before(l.started.Add(adminLoginWindow)) {
		l.started, l.total, l.peers = now, 0, make(map[string]int)
	}
	if l.total >= adminLoginGlobalLimit || l.peers[peer] >= adminLoginIPLimit {
		return l.started.Add(adminLoginWindow).Sub(now)
	}
	l.total++
	l.peers[peer]++
	return 0
}

func adminLoginPeer(remoteAddr string) string {
	// Do not trust X-Forwarded-For or X-Real-IP supplied by arbitrary clients.
	// Behind an ingress, all clients share that peer's conservative limit.
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return "unknown"
}
