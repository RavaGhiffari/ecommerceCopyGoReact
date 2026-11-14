package addresses

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// List returns the caller's addresses, default first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Address, error) {
	return NewRepository(s.pool).List(ctx, userID)
}

// Get returns one of the caller's addresses (checkout uses it too).
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Address, error) {
	return NewRepository(s.pool).Get(ctx, userID, id)
}

// Every write below runs in one transaction that starts with LockUser.
//
// Why the lock: "count < 10, then insert" and "clear the old default, then
// set the new one" are each two statements. Without the lock, two requests of
// the same user could both count 9 and both insert (11 addresses), or both
// clear and then both try to set a default (one would fail on the unique
// index with a 500). With the lock the second request waits and then sees
// the first one's result.

// Create adds an address. The first address is always the default; a later
// one becomes the default only when makeDefault is true.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, f Fields, makeDefault bool) (Address, error) {
	f, err := f.Clean()
	if err != nil {
		return Address{}, err
	}

	var created Address
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		if err := repo.LockUser(ctx, userID); err != nil {
			return err
		}
		n, err := repo.Count(ctx, userID)
		if err != nil {
			return err
		}
		if n >= MaxPerUser {
			return ErrLimitReached
		}
		isDefault := makeDefault || n == 0
		if isDefault && n > 0 {
			// Clear first: the partial unique index never allows two defaults,
			// not even for a moment inside the transaction.
			if err := repo.ClearDefault(ctx, userID); err != nil {
				return err
			}
		}
		created, err = repo.Create(ctx, userID, f, isDefault)
		return err
	})
	return created, err
}

// UpdateInput: nil leaves a field unchanged. The default flag is not part of
// it; use SetDefault, so "un-defaulting" the only default is impossible.
type UpdateInput struct {
	Label         *string
	RecipientName *string
	Phone         *string
	AddressLine   *string
	City          *string
	Province      *string
	PostalCode    *string
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (Address, error) {
	var updated Address
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		if err := repo.LockUser(ctx, userID); err != nil {
			return err
		}
		cur, err := repo.Get(ctx, userID, id)
		if err != nil {
			return err
		}
		f := Fields{
			Label: pick(in.Label, cur.Label), RecipientName: pick(in.RecipientName, cur.RecipientName),
			Phone: pick(in.Phone, cur.Phone), AddressLine: pick(in.AddressLine, cur.AddressLine),
			City: pick(in.City, cur.City), Province: pick(in.Province, cur.Province),
			PostalCode: pick(in.PostalCode, cur.PostalCode),
		}
		if f, err = f.Clean(); err != nil {
			return err
		}
		updated, err = repo.Update(ctx, userID, id, f)
		return err
	})
	return updated, err
}

// Delete removes an address. If it was the default, the newest remaining
// address becomes the default, so a user with addresses always has one.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		if err := repo.LockUser(ctx, userID); err != nil {
			return err
		}
		wasDefault, err := repo.Delete(ctx, userID, id)
		if err != nil || !wasDefault {
			return err
		}
		return repo.PromoteNewest(ctx, userID)
	})
}

// SetDefault makes one of the caller's addresses the default.
func (s *Service) SetDefault(ctx context.Context, userID, id uuid.UUID) (Address, error) {
	var a Address
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		if err := repo.LockUser(ctx, userID); err != nil {
			return err
		}
		cur, err := repo.Get(ctx, userID, id) // 404 before touching the current default
		if err != nil {
			return err
		}
		if cur.IsDefault {
			a = cur
			return nil
		}
		if err := repo.ClearDefault(ctx, userID); err != nil {
			return err
		}
		a, err = repo.SetDefault(ctx, userID, id)
		return err
	})
	return a, err
}

func pick(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}
