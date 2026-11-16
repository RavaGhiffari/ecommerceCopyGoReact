// Package admin is the platform back office (Phase 6d, PLAN.md §6.0 Q9–Q13):
// users and stores (list, search, suspend, reactivate), platform stats and
// the audit log. Platform orders live in the orders module and review
// moderation in the reviews module, next to the code that owns that data;
// all of them write their audit rows through the audit package.
//
// Authorization: every route is behind RequireRole(admin) in the router. The
// rules below are the ones a role check cannot express (an admin cannot
// suspend themselves or another admin).
package admin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/audit"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/orders"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrStoreNotFound = errors.New("store not found")
	ErrSelf          = errors.New("admins cannot change their own status")
	ErrTargetAdmin   = errors.New("admins cannot change another admin's status")
)

// Account and store statuses (CHECK constraints of 000002 and 000006).
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

var accountStatuses = []string{StatusActive, StatusSuspended}

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
	MaxSearchLength = 100
	StatsWindow     = 30 * 24 * time.Hour
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ---- shared request checks ---------------------------------------------------------

// Paging applies the defaults (0 = default) and records invalid values.
func Paging(errs validation.Errors, page, pageSize int) (int, int) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if page < 1 {
		errs.Add("page", "must be 1 or more")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		errs.Add("pageSize", fmt.Sprintf("must be between 1 and %d", MaxPageSize))
	}
	return page, pageSize
}

// searchPattern turns the search box text into an ILIKE pattern, or nil for
// "no search". % _ and \ are escaped so they match themselves, not anything.
func searchPattern(errs validation.Errors, q string) *string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	if utf8.RuneCountInString(q) > MaxSearchLength {
		errs.Add("q", fmt.Sprintf("must be at most %d characters", MaxSearchLength))
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	p := "%" + escaped + "%"
	return &p
}

// statusFilter validates an optional ?status= ("" = any).
func statusFilter(errs validation.Errors, status string, allowed []string) *string {
	if status == "" {
		return nil
	}
	if !slices.Contains(allowed, status) {
		errs.Add("status", "must be one of "+strings.Join(allowed, ", "))
	}
	return &status
}

// newStatus validates the body of a status change.
func newStatus(status string, allowed []string) error {
	errs := validation.Errors{}
	if status == "" {
		errs.Add("status", "is required")
	} else if !slices.Contains(allowed, status) {
		errs.Add("status", "must be one of "+strings.Join(allowed, ", "))
	}
	return errs.Err()
}

// Page is one page of any admin list.
type Page[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

func newPage[T any](items []T, page, pageSize int, total int64) Page[T] {
	return Page[T]{Items: items, Page: page, PageSize: pageSize, Total: total,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize))}
}

// ---- users ---------------------------------------------------------------------------

type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Status    string
	Roles     []string
	CreatedAt time.Time
}

// ListUsers searches by email or name; status "" = any.
func (s *Service) ListUsers(ctx context.Context, q, status string, page, pageSize int) (Page[User], error) {
	errs := validation.Errors{}
	page, pageSize = Paging(errs, page, pageSize)
	pattern := searchPattern(errs, q)
	filter := statusFilter(errs, status, accountStatuses)
	if err := errs.Err(); err != nil {
		return Page[User]{}, err
	}

	db := queries.New(s.pool)
	total, err := db.CountAdminUsers(ctx, queries.CountAdminUsersParams{Status: filter, Q: pattern})
	if err != nil {
		return Page[User]{}, fmt.Errorf("count users: %w", err)
	}
	rows, err := db.ListAdminUsers(ctx, queries.ListAdminUsersParams{
		Status: filter, Q: pattern, PageLimit: int32(pageSize), PageOffset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return Page[User]{}, fmt.Errorf("list users: %w", err)
	}
	items := make([]User, len(rows))
	for i, r := range rows {
		items[i] = User{ID: r.ID, Email: r.Email, Name: r.Name, Status: r.Status, Roles: r.Roles, CreatedAt: r.CreatedAt}
	}
	return newPage(items, page, pageSize, total), nil
}

// SetUserStatus suspends or reactivates a user, in one transaction:
//
//  1. Validate the status (422).
//  2. Lock the target user row (unknown → 404).
//  3. Rules: not yourself, not another admin (409). Checked under the lock.
//  4. Same status as now → nothing changes and no audit row is written
//     (repeating a request is harmless).
//  5. Update the status; on suspend also delete every session of the user,
//     so they are logged out now and stay logged out after a reactivation.
//  6. Audit row with the user's email, in the same transaction.
//
// Suspending a user does not suspend their store (PLAN.md Q10).
func (s *Service) SetUserStatus(ctx context.Context, actorID, userID uuid.UUID, status string) (User, error) {
	if err := newStatus(status, accountStatuses); err != nil {
		return User{}, err
	}
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		u, err := q.LockUserForAdmin(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		if err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		if u.ID == actorID {
			return ErrSelf
		}
		if u.IsAdmin {
			return ErrTargetAdmin
		}
		if u.Status == status {
			return nil
		}

		if err := q.SetUserStatus(ctx, queries.SetUserStatusParams{ID: userID, Status: status}); err != nil {
			return fmt.Errorf("set user status: %w", err)
		}
		action := audit.UserReactivate
		details := map[string]any{"email": u.Email}
		if status == StatusSuspended {
			action = audit.UserSuspend
			ended, err := q.DeleteUserSessions(ctx, userID)
			if err != nil {
				return fmt.Errorf("end sessions: %w", err)
			}
			details["sessionsEnded"] = ended
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: actorID, Action: action, TargetType: audit.TargetUser, TargetID: userID, Details: details,
		})
	})
	if err != nil {
		return User{}, err
	}
	return s.user(ctx, userID)
}

func (s *Service) user(ctx context.Context, id uuid.UUID) (User, error) {
	q := queries.New(s.pool)
	row, err := q.GetUserByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	roles, err := q.ListUserRoleNames(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("user roles: %w", err)
	}
	return User{ID: row.ID, Email: row.Email, Name: row.Name, Status: row.Status, Roles: roles, CreatedAt: row.CreatedAt}, nil
}

// ---- stores --------------------------------------------------------------------------

type Store struct {
	ID                 uuid.UUID
	Name               string
	Slug               string
	Location           string
	Status             string
	CreatedAt          time.Time
	OwnerID            uuid.UUID
	OwnerName          string
	OwnerEmail         string
	ActiveProductCount int32
}

// ListStores searches by store name, slug or owner email; status "" = any.
func (s *Service) ListStores(ctx context.Context, q, status string, page, pageSize int) (Page[Store], error) {
	errs := validation.Errors{}
	page, pageSize = Paging(errs, page, pageSize)
	pattern := searchPattern(errs, q)
	filter := statusFilter(errs, status, accountStatuses)
	if err := errs.Err(); err != nil {
		return Page[Store]{}, err
	}

	db := queries.New(s.pool)
	total, err := db.CountAdminStores(ctx, queries.CountAdminStoresParams{Status: filter, Q: pattern})
	if err != nil {
		return Page[Store]{}, fmt.Errorf("count stores: %w", err)
	}
	rows, err := db.ListAdminStores(ctx, queries.ListAdminStoresParams{
		Status: filter, Q: pattern, PageLimit: int32(pageSize), PageOffset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return Page[Store]{}, fmt.Errorf("list stores: %w", err)
	}
	items := make([]Store, len(rows))
	for i, r := range rows {
		items[i] = Store(r)
	}
	return newPage(items, page, pageSize, total), nil
}

// SetStoreStatus suspends or reactivates a store: lock → same status is a
// no-op → update → audit row, one transaction. A suspended store leaves the
// catalog and its cart lines are flagged (both already check stores.status);
// its existing orders keep their lifecycle (PLAN.md Q10).
func (s *Service) SetStoreStatus(ctx context.Context, actorID, storeID uuid.UUID, status string) (Store, error) {
	if err := newStatus(status, accountStatuses); err != nil {
		return Store{}, err
	}
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := queries.New(tx)
		st, err := q.LockStoreForAdmin(ctx, storeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStoreNotFound
		}
		if err != nil {
			return fmt.Errorf("lock store: %w", err)
		}
		if st.Status == status {
			return nil
		}
		if err := q.SetStoreStatus(ctx, queries.SetStoreStatusParams{ID: storeID, Status: status}); err != nil {
			return fmt.Errorf("set store status: %w", err)
		}
		action := audit.StoreReactivate
		if status == StatusSuspended {
			action = audit.StoreSuspend
		}
		return audit.Write(ctx, tx, audit.Entry{
			ActorID: actorID, Action: action, TargetType: audit.TargetStore, TargetID: storeID,
			Details: map[string]any{"name": st.Name, "slug": st.Slug},
		})
	})
	if err != nil {
		return Store{}, err
	}
	return s.store(ctx, storeID)
}

func (s *Service) store(ctx context.Context, id uuid.UUID) (Store, error) {
	row, err := queries.New(s.pool).GetAdminStore(ctx, id)
	if err != nil {
		return Store{}, fmt.Errorf("get store: %w", err)
	}
	return Store(row), nil
}

// ---- stats ---------------------------------------------------------------------------

type Stats struct {
	Since          time.Time
	UsersByStatus  map[string]int64
	StoresByStatus map[string]int64
	PublicProducts int64
	OrdersByStatus map[string]int64
	GMVAll         decimal.Decimal
	GMVWindow      decimal.Decimal
}

// Stats are platform-wide counts, computed on read like the seller dashboard.
// GMV uses the seller revenue rule (orders.RevenueStatuses, items only).
func (s *Service) Stats(ctx context.Context, now time.Time) (Stats, error) {
	q := queries.New(s.pool)
	st := Stats{
		Since:          now.Add(-StatsWindow),
		UsersByStatus:  map[string]int64{StatusActive: 0, StatusSuspended: 0},
		StoresByStatus: map[string]int64{StatusActive: 0, StatusSuspended: 0},
		OrdersByStatus: map[string]int64{},
	}
	for _, o := range orders.Statuses {
		st.OrdersByStatus[o] = 0
	}

	users, err := q.AdminCountUsersByStatus(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("count users: %w", err)
	}
	for _, r := range users {
		st.UsersByStatus[r.Status] = r.N
	}
	storeRows, err := q.AdminCountStoresByStatus(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("count stores: %w", err)
	}
	for _, r := range storeRows {
		st.StoresByStatus[r.Status] = r.N
	}
	if st.PublicProducts, err = q.AdminCountPublicProducts(ctx); err != nil {
		return Stats{}, fmt.Errorf("count products: %w", err)
	}
	orderRows, err := q.AdminCountOrdersByStatus(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("count orders: %w", err)
	}
	for _, r := range orderRows {
		st.OrdersByStatus[r.Status] = r.N
	}
	gmv, err := q.AdminGMV(ctx, queries.AdminGMVParams{Since: st.Since, Statuses: orders.RevenueStatuses})
	if err != nil {
		return Stats{}, fmt.Errorf("gmv: %w", err)
	}
	st.GMVAll, st.GMVWindow = gmv.GmvAllTime, gmv.GmvWindow
	return st, nil
}

// ---- audit log -----------------------------------------------------------------------

func (s *Service) AuditLogs(ctx context.Context, page, pageSize int) (Page[audit.Log], error) {
	errs := validation.Errors{}
	page, pageSize = Paging(errs, page, pageSize)
	if err := errs.Err(); err != nil {
		return Page[audit.Log]{}, err
	}
	items, total, err := audit.List(ctx, s.pool, int32(pageSize), int32((page-1)*pageSize))
	if err != nil {
		return Page[audit.Log]{}, err
	}
	return newPage(items, page, pageSize, total), nil
}
