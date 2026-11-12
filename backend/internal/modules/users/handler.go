package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetMe handles GET /api/v1/me. The route sits behind RequireAuth, so a
// principal is always present. The user ID comes from the session, never from
// the URL or body: there is no way to ask for someone else's profile here.
func (h *Handler) GetMe(c *gin.Context) {
	p, _ := authctx.FromContext(c.Request.Context())

	u, err := h.svc.GetProfile(c.Request.Context(), p.UserID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewUserResponse(u), "")
}

// UpdateMe handles PATCH /api/v1/me.
func (h *Handler) UpdateMe(c *gin.Context) {
	p, _ := authctx.FromContext(c.Request.Context())

	var body UpdateProfileRequest
	if !request.BindJSON(c, &body) {
		return
	}

	u, err := h.svc.UpdateProfile(c.Request.Context(), p.UserID, UpdateProfileInput(body))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewUserResponse(u), "Profile updated")
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrNotFound):
		// The session is valid but the user row is gone: treat as logged out.
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthenticated, "Authentication required", nil)
	default:
		response.Internal(c, err)
	}
}
