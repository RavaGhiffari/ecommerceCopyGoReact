package stores

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/slug"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Service struct {
	pool *pgxpool.Pool
	repo *Repository
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, repo: NewRepository(pool)}
}

type OpenInput struct {
	Name        string
	Slug        string // optional: derived from Name
	Description string
	Location    string
}

// Open creates the caller's store and makes them a seller.
//
// Both happen in ONE transaction: a store whose owner lacks the seller role
// (or a seller role without a store) would be a broken state that later code
// would have to defend against everywhere.
//
// The owner comes from the session (ownerID), never from the request body.
func (s *Service) Open(ctx context.Context, ownerID uuid.UUID, in OpenInput) (Store, error) {
	errs := validation.Errors{}
	st := Store{
		OwnerID:     ownerID,
		Name:        cleanText(errs, "name", in.Name, 3, 100),
		Description: cleanText(errs, "description", in.Description, 0, 1000),
		Location:    cleanText(errs, "location", in.Location, 2, 100),
	}
	st.Slug = strings.TrimSpace(in.Slug)
	if st.Slug == "" {
		st.Slug = slug.Make(st.Name)
	}
	if !slug.Valid(st.Slug) {
		errs.Add("slug", "may only contain a-z, 0-9 and single hyphens")
	}
	if err := errs.Err(); err != nil {
		return Store{}, err
	}

	var created Store
	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		if created, err = NewRepository(tx).Create(ctx, st); err != nil {
			return err
		}
		return users.NewRepository(tx).EnsureRole(ctx, ownerID, authctx.RoleSeller)
	})
	return created, err
}

// Mine returns the caller's store, or ErrNotFound.
func (s *Service) Mine(ctx context.Context, ownerID uuid.UUID) (Store, error) {
	return s.repo.GetByOwner(ctx, ownerID)
}

// UpdateInput: nil leaves a field unchanged. The slug never changes after
// creation, because it is part of every product URL of the store.
type UpdateInput struct {
	Name        *string
	Description *string
	Location    *string
}

func (s *Service) UpdateMine(ctx context.Context, ownerID uuid.UUID, in UpdateInput) (Store, error) {
	st, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		return Store{}, err
	}
	errs := validation.Errors{}
	if in.Name != nil {
		st.Name = cleanText(errs, "name", *in.Name, 3, 100)
	}
	if in.Description != nil {
		st.Description = cleanText(errs, "description", *in.Description, 0, 1000)
	}
	if in.Location != nil {
		st.Location = cleanText(errs, "location", *in.Location, 2, 100)
	}
	if err := errs.Err(); err != nil {
		return Store{}, err
	}
	return s.repo.Update(ctx, st)
}

// Public returns an active store by slug for the public store page.
func (s *Service) Public(ctx context.Context, slug string) (Store, error) {
	return s.repo.GetActiveBySlug(ctx, slug)
}

func cleanText(errs validation.Errors, field, v string, minLen, maxLen int) string {
	v = strings.TrimSpace(v)
	switch n := utf8.RuneCountInString(v); {
	case n == 0 && minLen > 0:
		errs.Add(field, "is required")
	case n < minLen:
		errs.Add(field, "is too short")
	case n > maxLen:
		errs.Add(field, "is too long")
	}
	return v
}
