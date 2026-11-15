package products

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type Service struct {
	pool       *pgxpool.Pool
	repo       *Repository
	categories *categories.Repository
	stores     *stores.Repository
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:       pool,
		repo:       NewRepository(pool),
		categories: categories.NewRepository(pool),
		stores:     stores.NewRepository(pool),
	}
}

// PublicQuery is what the listing page asks for. Zero values mean defaults.
type PublicQuery struct {
	Q            string
	CategorySlug string
	StoreSlug    string
	Sort         string
	Page         int
	PageSize     int
}

// ListPublic returns one page of active products.
// Unknown category/store slugs are errors (404), not empty pages, so a typo
// in a URL is distinguishable from "no products yet".
func (s *Service) ListPublic(ctx context.Context, in PublicQuery) (Page, error) {
	if in.Sort == "" {
		in.Sort = SortPopular
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = DefaultPageSize
	}

	errs := validation.Errors{}
	switch in.Sort {
	case SortPopular, SortNewest, SortPriceAsc, SortPriceDesc, SortRating:
	default:
		errs.Add("sort", "must be one of popular, newest, price_asc, price_desc, rating")
	}
	if in.Page < 1 || in.Page > 10_000 {
		errs.Add("page", "must be between 1 and 10000")
	}
	if in.PageSize < 1 || in.PageSize > MaxPageSize {
		errs.Add("pageSize", "must be between 1 and 48")
	}
	if err := errs.Err(); err != nil {
		return Page{}, err
	}

	p := listParams{
		Search: buildSearchQuery(in.Q),
		Sort:   in.Sort,
		Limit:  int32(in.PageSize),
		Offset: int32((in.Page - 1) * in.PageSize),
	}

	if in.CategorySlug != "" {
		cat, err := s.categories.GetBySlug(ctx, in.CategorySlug)
		if errors.Is(err, categories.ErrNotFound) {
			return Page{}, ErrCategoryNotFound
		}
		if err != nil {
			return Page{}, err
		}
		// Browsing a parent includes its subcategories.
		if p.CategoryIDs, err = s.categories.TreeIDs(ctx, cat.ID); err != nil {
			return Page{}, err
		}
	}
	if in.StoreSlug != "" {
		st, err := s.stores.GetActiveBySlug(ctx, in.StoreSlug)
		if errors.Is(err, stores.ErrNotFound) {
			return Page{}, ErrStoreNotFound
		}
		if err != nil {
			return Page{}, err
		}
		p.StoreID = &st.ID
	}

	items, total, err := s.repo.ListPublic(ctx, p)
	if err != nil {
		return Page{}, err
	}
	// Past the last page there are no rows, so the window-function total is
	// missing; ask for one row of the first page just to learn it.
	if len(items) == 0 && in.Page > 1 {
		first := p
		first.Limit, first.Offset = 1, 0
		if _, total, err = s.repo.ListPublic(ctx, first); err != nil {
			return Page{}, err
		}
	}

	totalPages := int((total + int64(in.PageSize) - 1) / int64(in.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}
	return Page{Items: items, Page: in.Page, PageSize: in.PageSize, Total: total, TotalPages: totalPages}, nil
}

// PublicDetail is the product page at /stores/{storeSlug}/products/{productSlug}.
func (s *Service) PublicDetail(ctx context.Context, storeSlug, productSlug string) (Detail, error) {
	row, err := s.repo.GetPublic(ctx, storeSlug, productSlug)
	if err != nil {
		return Detail{}, err
	}
	variants, err := s.repo.Variants(ctx, row.ID)
	if err != nil {
		return Detail{}, err
	}
	if len(variants) == 0 {
		return Detail{}, ErrNotFound // same rule as the listing: no variant, not public
	}
	images, err := s.repo.Images(ctx, row.ID)
	if err != nil {
		return Detail{}, err
	}
	cat, err := s.categories.GetByID(ctx, row.CategoryID)
	if err != nil {
		return Detail{}, err
	}

	d := Detail{
		Summary: Summary{
			ID: row.ID, Slug: row.Slug, Name: row.Name, CategoryID: row.CategoryID,
			SoldCount: row.SoldCount, CreatedAt: row.CreatedAt,
			RatingCount: row.RatingCount, RatingTotal: row.RatingTotal,
			Store: StoreRef{ID: row.StoreID, Slug: row.StoreSlug, Name: row.StoreName, Location: row.StoreLocation},
		},
		Description: row.Description,
		Category:    cat,
		Images:      images,
		Variants:    variants,
	}
	if len(images) > 0 {
		d.Image = &images[0]
	}
	d.LowestPrice, d.HighestPrice = variants[0].Price, variants[0].Price
	for _, v := range variants {
		if v.Price.LessThan(d.LowestPrice) {
			d.LowestPrice = v.Price
		}
		if v.Price.GreaterThan(d.HighestPrice) {
			d.HighestPrice = v.Price
		}
		if v.Stock > 0 {
			d.InStock = true
		}
	}
	return d, nil
}

// Related returns other popular products from the same category.
func (s *Service) Related(ctx context.Context, productID uuid.UUID, limit int) ([]Summary, error) {
	categoryID, err := s.repo.PublicCategoryOf(ctx, productID)
	if err != nil {
		return nil, err
	}
	items, _, err := s.repo.ListPublic(ctx, listParams{
		CategoryIDs: []uuid.UUID{categoryID},
		ExcludeID:   &productID,
		Sort:        SortPopular,
		Limit:       int32(limit),
	})
	return items, err
}
