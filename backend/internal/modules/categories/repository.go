package categories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// All sqlc row types for categories have the same five columns, so they
// convert to ListCategoriesRow and share one mapper.
func toCategory(r queries.ListCategoriesRow) Category {
	return Category{ID: r.ID, ParentID: r.ParentID, Name: r.Name, Slug: r.Slug, SortOrder: r.SortOrder}
}

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.q.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	out := make([]Category, len(rows))
	for i, row := range rows {
		out[i] = toCategory(row)
	}
	return out, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Category, error) {
	row, err := r.q.GetCategoryByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return toCategory(queries.ListCategoriesRow(row)), nil
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (Category, error) {
	row, err := r.q.GetCategoryBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category by slug: %w", err)
	}
	return toCategory(queries.ListCategoriesRow(row)), nil
}

func (r *Repository) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := r.q.CategoryHasChildren(ctx, id)
	if err != nil {
		return false, fmt.Errorf("category has children: %w", err)
	}
	return has, nil
}

// TreeIDs returns the category and its children (for "browse Elektronik").
func (r *Repository) TreeIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q.ListCategoryTreeIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list category tree: %w", err)
	}
	return ids, nil
}

func (r *Repository) Create(ctx context.Context, c Category) (Category, error) {
	row, err := r.q.CreateCategory(ctx, queries.CreateCategoryParams{
		ParentID: c.ParentID, Name: c.Name, Slug: c.Slug, SortOrder: c.SortOrder,
	})
	if err != nil {
		return Category{}, translate(err, "create category")
	}
	return toCategory(queries.ListCategoriesRow(row)), nil
}

func (r *Repository) Update(ctx context.Context, c Category) (Category, error) {
	row, err := r.q.UpdateCategory(ctx, queries.UpdateCategoryParams{
		ID: c.ID, ParentID: c.ParentID, Name: c.Name, Slug: c.Slug, SortOrder: c.SortOrder,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	if err != nil {
		return Category{}, translate(err, "update category")
	}
	return toCategory(queries.ListCategoriesRow(row)), nil
}

func translate(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "categories_slug_key":
			return ErrSlugTaken
		case "categories_max_two_levels":
			return ErrTooDeep
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
