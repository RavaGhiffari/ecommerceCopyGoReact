package auth

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/middleware"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/ratelimit"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	CodeEmailTaken         = "EMAIL_TAKEN"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeAccountSuspended   = "ACCOUNT_SUSPENDED"
)

// Limits for brute-force protection. Tests pass larger values.
type Limits struct {
	LoginAttempts    int // consecutive failures allowed per IP + email
	LoginWindow      time.Duration
	RegisterAttempts int // sign-ups allowed per IP
	RegisterWindow   time.Duration
}

var DefaultLimits = Limits{
	LoginAttempts:    5,
	LoginWindow:      15 * time.Minute,
	RegisterAttempts: 10,
	RegisterWindow:   time.Hour,
}

type Handler struct {
	svc          *Service
	secureCookie bool // true in production: cookie only sent over HTTPS
	login        *ratelimit.Limiter
	register     *ratelimit.Limiter
}

func NewHandler(svc *Service, secureCookie bool, limits Limits) *Handler {
	return &Handler{
		svc:          svc,
		secureCookie: secureCookie,
		login:        ratelimit.New(limits.LoginAttempts, limits.LoginWindow),
		register:     ratelimit.New(limits.RegisterAttempts, limits.RegisterWindow),
	}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(c *gin.Context) {
	if ok, retry := h.register.Allow("ip:" + c.ClientIP()); !ok {
		rateLimited(c, retry)
		return
	}

	var body RegisterRequest
	if !request.BindJSON(c, &body) {
		return
	}

	u, sess, err := h.svc.Register(c.Request.Context(), RegisterInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	h.setSessionCookie(c, sess)
	response.OK(c, http.StatusCreated, users.NewUserResponse(u), "Account created")
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var body LoginRequest
	if !request.BindJSON(c, &body) {
		return
	}

	// Per IP + email: one attacker cannot lock out a victim from everywhere,
	// and one IP cannot hammer a single account.
	key := c.ClientIP() + "|" + users.NormalizeEmail(body.Email)
	if ok, retry := h.login.Allow(key); !ok {
		rateLimited(c, retry)
		return
	}

	previous, _ := c.Cookie(middleware.SessionCookieName)
	u, sess, err := h.svc.Login(c.Request.Context(), body.Email, body.Password, previous)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.login.Reset(key) // only consecutive failures count
	h.setSessionCookie(c, sess)
	response.OK(c, http.StatusOK, users.NewUserResponse(u), "Logged in")
}

// Logout handles POST /api/v1/auth/logout. Idempotent: logging out without a
// session still succeeds.
func (h *Handler) Logout(c *gin.Context) {
	raw, _ := c.Cookie(middleware.SessionCookieName)
	if err := h.svc.Logout(c.Request.Context(), raw); err != nil {
		response.Internal(c, err)
		return
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) setSessionCookie(c *gin.Context, s Session) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    s.Token,
		Path:     "/",
		Expires:  s.ExpiresAt,
		MaxAge:   int(time.Until(s.ExpiresAt).Seconds()),
		HttpOnly: true,                 // JavaScript cannot read it, so XSS cannot steal it
		Secure:   h.secureCookie,       // HTTPS only in production
		SameSite: http.SameSiteLaxMode, // not sent on cross-site POSTs (CSRF)
	})
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, users.ErrEmailTaken):
		response.Fail(c, http.StatusConflict, CodeEmailTaken, "This email is already registered",
			map[string]string{"email": "is already registered"})
	case errors.Is(err, ErrInvalidCredentials):
		response.Fail(c, http.StatusUnauthorized, CodeInvalidCredentials, "Incorrect email or password", nil)
	case errors.Is(err, ErrAccountSuspended):
		response.Fail(c, http.StatusForbidden, CodeAccountSuspended, "This account has been suspended", nil)
	default:
		response.Internal(c, err)
	}
}

func rateLimited(c *gin.Context, retryAfter time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
	response.Fail(c, http.StatusTooManyRequests, response.CodeRateLimited,
		"Too many attempts. Please try again later.", nil)
}
