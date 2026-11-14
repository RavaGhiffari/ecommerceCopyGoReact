package addresses

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

// Repository works with a pool or a transaction (queries.DBTX). It holds no
// rules; the service decides what runs together in one transaction.
type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

func toAddress(r queries.UserAddress) Address {
	return Address{
		ID: r.ID, UserID: r.UserID, Label: r.Label, RecipientName: r.RecipientName, Phone: r.Phone,
		AddressLine: r.AddressLine, City: r.City, Province: r.Province, PostalCode: r.PostalCode,
		IsDefault: r.IsDefault, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// LockUser takes the per-user lock that serialises address changes. Must be
// called inside a transaction; the lock is released at commit/rollback.
func (r *Repository) LockUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.q.LockUserForAddressChange(ctx, userID); err != nil {
		return fmt.Errorf("lock user for address change: %w", err)
	}
	return nil
}

func (r *Repository) Count(ctx context.Context, userID uuid.UUID) (int32, error) {
	n, err := r.q.CountUserAddresses(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count addresses: %w", err)
	}
	return n, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]Address, error) {
	rows, err := r.q.ListUserAddresses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	out := make([]Address, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAddress(row))
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, userID, id uuid.UUID) (Address, error) {
	row, err := r.q.GetUserAddress(ctx, queries.GetUserAddressParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Address{}, ErrNotFound
	}
	if err != nil {
		return Address{}, fmt.Errorf("get address: %w", err)
	}
	return toAddress(row), nil
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, f Fields, isDefault bool) (Address, error) {
	row, err := r.q.CreateUserAddress(ctx, queries.CreateUserAddressParams{
		UserID: userID, Label: f.Label, RecipientName: f.RecipientName, Phone: f.Phone,
		AddressLine: f.AddressLine, City: f.City, Province: f.Province, PostalCode: f.PostalCode,
		IsDefault: isDefault,
	})
	if err != nil {
		return Address{}, fmt.Errorf("create address: %w", err)
	}
	return toAddress(row), nil
}

func (r *Repository) Update(ctx context.Context, userID, id uuid.UUID, f Fields) (Address, error) {
	row, err := r.q.UpdateUserAddress(ctx, queries.UpdateUserAddressParams{
		ID: id, UserID: userID, Label: f.Label, RecipientName: f.RecipientName, Phone: f.Phone,
		AddressLine: f.AddressLine, City: f.City, Province: f.Province, PostalCode: f.PostalCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Address{}, ErrNotFound
	}
	if err != nil {
		return Address{}, fmt.Errorf("update address: %w", err)
	}
	return toAddress(row), nil
}

// Delete removes the address and reports whether it was the default.
func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) (wasDefault bool, err error) {
	wasDefault, err = r.q.DeleteUserAddress(ctx, queries.DeleteUserAddressParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("delete address: %w", err)
	}
	return wasDefault, nil
}

func (r *Repository) ClearDefault(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.ClearDefaultUserAddress(ctx, userID); err != nil {
		return fmt.Errorf("clear default address: %w", err)
	}
	return nil
}

func (r *Repository) SetDefault(ctx context.Context, userID, id uuid.UUID) (Address, error) {
	row, err := r.q.SetDefaultUserAddress(ctx, queries.SetDefaultUserAddressParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Address{}, ErrNotFound
	}
	if err != nil {
		return Address{}, fmt.Errorf("set default address: %w", err)
	}
	return toAddress(row), nil
}

func (r *Repository) PromoteNewest(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.PromoteNewestUserAddress(ctx, userID); err != nil {
		return fmt.Errorf("promote newest address: %w", err)
	}
	return nil
}
