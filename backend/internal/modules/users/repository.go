package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Repository wraps the sqlc queries for users. It translates database
// outcomes (no rows, unique violation) into this package's errors so the
// layers above never inspect PostgreSQL error codes.
type Repository struct {
	q *queries.Queries
}

// NewRepository works with the pool or with a transaction (both are DBTX).
func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

const uniqueViolation = "23505" // PostgreSQL SQLSTATE

func (r *Repository) Create(ctx context.Context, email, passwordHash, name string) (User, error) {
	row, err := r.q.CreateUser(ctx, queries.CreateUserParams{Email: email, PasswordHash: passwordHash, Name: name})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == "users_email_lower_key" {
			// The database is the final judge of uniqueness: checking first
			// with a SELECT would race with a concurrent sign-up.
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return fromRow(queries.GetUserByIDRow(row)), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return fromRow(row), nil
}

func (r *Repository) UpdateProfile(ctx context.Context, id uuid.UUID, name string, phone *string) (User, error) {
	row, err := r.q.UpdateUserProfile(ctx, queries.UpdateUserProfileParams{ID: id, Name: name, Phone: phone})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}
	return fromRow(queries.GetUserByIDRow(row)), nil
}

func (r *Repository) AssignRole(ctx context.Context, userID uuid.UUID, role string) error {
	n, err := r.q.AssignRoleByName(ctx, queries.AssignRoleByNameParams{UserID: userID, RoleName: role})
	if err != nil {
		return fmt.Errorf("assign role %q: %w", role, err)
	}
	if n == 0 {
		return fmt.Errorf("assign role: role %q does not exist", role)
	}
	return nil
}

// EnsureRole grants a role; already having it is not an error.
func (r *Repository) EnsureRole(ctx context.Context, userID uuid.UUID, role string) error {
	if err := r.q.EnsureRoleByName(ctx, queries.EnsureRoleByNameParams{UserID: userID, RoleName: role}); err != nil {
		return fmt.Errorf("ensure role %q: %w", role, err)
	}
	return nil
}

func (r *Repository) RoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	roles, err := r.q.ListUserRoleNames(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	return roles, nil
}
