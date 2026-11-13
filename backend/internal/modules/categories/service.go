package categories

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/slug"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Service struct {
	repo *Repository
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(pool)}
}

func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.repo.List(ctx)
}

// Input is the full definition of a category (create and replace).
type Input struct {
	Name      string
	Slug      string // optional: derived from Name when empty
	ParentID  *uuid.UUID
	SortOrder int32
}

func (s *Service) Create(ctx context.Context, in Input) (Category, error) {
	c, err := s.validate(ctx, uuid.Nil, in)
	if err != nil {
		return Category{}, err
	}
	return s.repo.Create(ctx, c)
}

// Replace overwrites a category with a new definition (PUT semantics).
func (s *Service) Replace(ctx context.Context, id uuid.UUID, in Input) (Category, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return Category{}, err
	}
	c, err := s.validate(ctx, id, in)
	if err != nil {
		return Category{}, err
	}
	c.ID = id
	return s.repo.Update(ctx, c)
}

// validate applies the rules; id is uuid.Nil for a new category.
func (s *Service) validate(ctx context.Context, id uuid.UUID, in Input) (Category, error) {
	errs := validation.Errors{}

	name := strings.TrimSpace(in.Name)
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		errs.Add("name", "is required")
	case n > 100:
		errs.Add("name", "must be at most 100 characters")
	}

	sl := strings.TrimSpace(in.Slug)
	if sl == "" {
		sl = slug.Make(name)
	}
	if !slug.Valid(sl) {
		errs.Add("slug", "may only contain a-z, 0-9 and single hyphens")
	}

	// Two levels: the parent must be top-level, and a category that has
	// children cannot itself be moved under a parent.
	if in.ParentID != nil {
		switch parent, err := s.repo.GetByID(ctx, *in.ParentID); {
		case errors.Is(err, ErrNotFound):
			errs.Add("parentId", "does not exist")
		case err != nil:
			return Category{}, err
		case parent.ID == id:
			errs.Add("parentId", "cannot be the category itself")
		case parent.ParentID != nil:
			errs.Add("parentId", "must be a top-level category (only two levels are allowed)")
		}
		if id != uuid.Nil {
			hasChildren, err := s.repo.HasChildren(ctx, id)
			if err != nil {
				return Category{}, err
			}
			if hasChildren {
				errs.Add("parentId", "a category with subcategories cannot get a parent")
			}
		}
	}

	if err := errs.Err(); err != nil {
		return Category{}, err
	}
	return Category{ParentID: in.ParentID, Name: name, Slug: sl, SortOrder: in.SortOrder}, nil
}
