package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/audit"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/request"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

// Every route here is behind RequireRole(admin) (see router.go). The JSON
// contract is mirrored in frontend/src/types/admin.ts (slice 6e).

const (
	CodeUserNotFound  = "USER_NOT_FOUND"
	CodeStoreNotFound = "STORE_NOT_FOUND"
	CodeSelf          = "CANNOT_CHANGE_SELF"
	CodeTargetAdmin   = "CANNOT_CHANGE_ADMIN"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// StatusRequest is the body of PATCH /admin/users/:id/status and
// PATCH /admin/stores/:id/status.
type StatusRequest struct {
	Status string `json:"status"` // active or suspended
}

// ---- responses ---------------------------------------------------------------------

type PageResponse[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

func newPageResponse[T, R any](p Page[T], conv func(T) R) PageResponse[R] {
	items := make([]R, len(p.Items))
	for i, it := range p.Items {
		items[i] = conv(it)
	}
	return PageResponse[R]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: p.Total, TotalPages: p.TotalPages}
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewUserResponse(u User) UserResponse {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return UserResponse{ID: u.ID.String(), Email: u.Email, Name: u.Name, Status: u.Status, Roles: roles, CreatedAt: u.CreatedAt.UTC()}
}

type OwnerResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type StoreResponse struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Slug               string        `json:"slug"`
	Location           string        `json:"location"`
	Status             string        `json:"status"`
	Owner              OwnerResponse `json:"owner"`
	ActiveProductCount int32         `json:"activeProductCount"`
	CreatedAt          time.Time     `json:"createdAt"`
}

func NewStoreResponse(s Store) StoreResponse {
	return StoreResponse{
		ID: s.ID.String(), Name: s.Name, Slug: s.Slug, Location: s.Location, Status: s.Status,
		Owner:              OwnerResponse{ID: s.OwnerID.String(), Name: s.OwnerName, Email: s.OwnerEmail},
		ActiveProductCount: s.ActiveProductCount, CreatedAt: s.CreatedAt.UTC(),
	}
}

type StatsResponse struct {
	Users          map[string]int64 `json:"users"`  // active, suspended
	Stores         map[string]int64 `json:"stores"` // active, suspended
	PublicProducts int64            `json:"publicProducts"`
	OrdersByStatus map[string]int64 `json:"ordersByStatus"`
	GMV            struct {
		Last30Days string `json:"last30Days"`
		AllTime    string `json:"allTime"`
	} `json:"gmv"`
	Since time.Time `json:"since"`
}

func NewStatsResponse(s Stats) StatsResponse {
	r := StatsResponse{
		Users: s.UsersByStatus, Stores: s.StoresByStatus, PublicProducts: s.PublicProducts,
		OrdersByStatus: s.OrdersByStatus, Since: s.Since.UTC(),
	}
	r.GMV.Last30Days = s.GMVWindow.StringFixed(2)
	r.GMV.AllTime = s.GMVAll.StringFixed(2)
	return r
}

type AuditLogResponse struct {
	ID         string          `json:"id"`
	Action     string          `json:"action"`
	TargetType string          `json:"targetType"`
	TargetID   string          `json:"targetId"`
	Details    json.RawMessage `json:"details"`
	Actor      OwnerResponse   `json:"actor"`
	CreatedAt  time.Time       `json:"createdAt"`
}

func NewAuditLogResponse(l audit.Log) AuditLogResponse {
	return AuditLogResponse{
		ID: l.ID.String(), Action: l.Action, TargetType: l.TargetType, TargetID: l.TargetID.String(),
		Details: l.Details, Actor: OwnerResponse{ID: l.ActorID.String(), Name: l.ActorName, Email: l.ActorEmail},
		CreatedAt: l.CreatedAt.UTC(),
	}
}

// ---- handlers ----------------------------------------------------------------------

// ListUsers handles GET /api/v1/admin/users?q=&status=&page=&pageSize=
func (h *Handler) ListUsers(c *gin.Context) {
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	p, err := h.svc.ListUsers(c.Request.Context(), c.Query("q"), c.Query("status"), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, newPageResponse(p, NewUserResponse), "")
}

// SetUserStatus handles PATCH /api/v1/admin/users/:id/status {status}
func (h *Handler) SetUserStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrUserNotFound)
		return
	}
	var body StatusRequest
	if !request.BindJSON(c, &body) {
		return
	}
	u, err := h.svc.SetUserStatus(c.Request.Context(), actorID(c), id, body.Status)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewUserResponse(u), "User status updated")
}

// ListStores handles GET /api/v1/admin/stores?q=&status=&page=&pageSize=
func (h *Handler) ListStores(c *gin.Context) {
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	p, err := h.svc.ListStores(c.Request.Context(), c.Query("q"), c.Query("status"), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, newPageResponse(p, NewStoreResponse), "")
}

// SetStoreStatus handles PATCH /api/v1/admin/stores/:id/status {status}
func (h *Handler) SetStoreStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.fail(c, ErrStoreNotFound)
		return
	}
	var body StatusRequest
	if !request.BindJSON(c, &body) {
		return
	}
	s, err := h.svc.SetStoreStatus(c.Request.Context(), actorID(c), id, body.Status)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewStoreResponse(s), "Store status updated")
}

// Stats handles GET /api/v1/admin/stats
func (h *Handler) Stats(c *gin.Context) {
	st, err := h.svc.Stats(c.Request.Context(), time.Now())
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, NewStatsResponse(st), "")
}

// AuditLogs handles GET /api/v1/admin/audit-logs?page=&pageSize=
func (h *Handler) AuditLogs(c *gin.Context) {
	page, pageSize, ok := h.paging(c)
	if !ok {
		return
	}
	p, err := h.svc.AuditLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.OK(c, http.StatusOK, newPageResponse(p, NewAuditLogResponse), "")
}

// ---- helpers -----------------------------------------------------------------------

func actorID(c *gin.Context) uuid.UUID {
	p, _ := authctx.FromContext(c.Request.Context())
	return p.UserID
}

// paging reads ?page= and ?pageSize= (0 = default); a non-number is a 422.
func (h *Handler) paging(c *gin.Context) (int, int, bool) {
	errs := validation.Errors{}
	page := IntQuery(c, errs, "page")
	pageSize := IntQuery(c, errs, "pageSize")
	if err := errs.Err(); err != nil {
		h.fail(c, err)
		return 0, 0, false
	}
	return page, pageSize, true
}

// IntQuery reads an optional whole-number query parameter.
func IntQuery(c *gin.Context, errs validation.Errors, name string) int {
	raw := c.Query(name)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		errs.Add(name, "must be a whole number")
	}
	return n
}

func (h *Handler) fail(c *gin.Context, err error) {
	var fields validation.Errors
	switch {
	case errors.As(err, &fields):
		response.ValidationFailed(c, fields)
	case errors.Is(err, ErrUserNotFound):
		response.Fail(c, http.StatusNotFound, CodeUserNotFound, "User not found", nil)
	case errors.Is(err, ErrStoreNotFound):
		response.Fail(c, http.StatusNotFound, CodeStoreNotFound, "Store not found", nil)
	case errors.Is(err, ErrSelf):
		response.Fail(c, http.StatusConflict, CodeSelf, "You cannot change your own account status", nil)
	case errors.Is(err, ErrTargetAdmin):
		response.Fail(c, http.StatusConflict, CodeTargetAdmin, "Another admin's account status cannot be changed", nil)
	default:
		response.Internal(c, err)
	}
}
