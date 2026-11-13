package products

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
)

type Repository struct {
	q *queries.Queries
}

func NewRepository(db queries.DBTX) *Repository {
	return &Repository{q: queries.New(db)}
}

// translate maps constraint violations to this package's errors. The database
// is the final judge of uniqueness; checking with a SELECT first would race
// with a concurrent request.
func translate(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "products_store_slug_key":
			return ErrSlugTaken
		case "product_variants_sku_key":
			return ErrSKUTaken
		case "product_variants_product_name_key":
			return ErrVariantNameTaken
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// ---- public -----------------------------------------------------------------

type listParams struct {
	CategoryIDs []uuid.UUID // nil = any category
	StoreID     *uuid.UUID
	ExcludeID   *uuid.UUID
	Search      *string
	Sort        string
	Limit       int32
	Offset      int32
}

func (r *Repository) ListPublic(ctx context.Context, p listParams) ([]Summary, int64, error) {
	rows, err := r.q.ListPublicProducts(ctx, queries.ListPublicProductsParams{
		CategoryIds: p.CategoryIDs,
		StoreID:     p.StoreID,
		ExcludeID:   p.ExcludeID,
		Search:      p.Search,
		Sort:        p.Sort,
		PageLimit:   p.Limit,
		PageOffset:  p.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list public products: %w", err)
	}

	var total int64
	items := make([]Summary, len(rows))
	for i, row := range rows {
		total = row.TotalCount // the same on every row (window function)
		items[i] = Summary{
			ID: row.ID, Slug: row.Slug, Name: row.Name, CategoryID: row.CategoryID,
			LowestPrice: row.MinPrice, HighestPrice: row.MaxPrice, SoldCount: row.SoldCount,
			InStock: row.InStock, CreatedAt: row.CreatedAt,
			Store: StoreRef{ID: row.StoreID, Slug: row.StoreSlug, Name: row.StoreName, Location: row.StoreLocation},
		}
		if row.ImageUrl != "" {
			items[i].Image = &Image{URL: row.ImageUrl, Alt: row.ImageAlt}
		}
	}
	return items, total, nil
}

func (r *Repository) GetPublic(ctx context.Context, storeSlug, productSlug string) (queries.GetPublicProductRow, error) {
	row, err := r.q.GetPublicProduct(ctx, queries.GetPublicProductParams{StoreSlug: storeSlug, ProductSlug: productSlug})
	if errors.Is(err, pgx.ErrNoRows) {
		return row, ErrNotFound
	}
	if err != nil {
		return row, fmt.Errorf("get public product: %w", err)
	}
	return row, nil
}

func (r *Repository) PublicCategoryOf(ctx context.Context, productID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.GetPublicProductCategory(ctx, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get product category: %w", err)
	}
	return id, nil
}

func (r *Repository) Variants(ctx context.Context, productID uuid.UUID) ([]Variant, error) {
	rows, err := r.q.ListVariantsWithStock(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	out := make([]Variant, len(rows))
	for i, v := range rows {
		out[i] = Variant{ID: v.ID, SKU: v.Sku, Name: v.Name, Price: v.Price, Stock: v.AvailableQuantity}
	}
	return out, nil
}

func (r *Repository) Images(ctx context.Context, productID uuid.UUID) ([]Image, error) {
	rows, err := r.q.ListProductImages(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	out := make([]Image, len(rows))
	for i, img := range rows {
		out[i] = Image{URL: img.Url, Alt: img.Alt}
	}
	return out, nil
}

// ---- seller -----------------------------------------------------------------

func toSellerProduct(r queries.GetStoreProductRow) SellerProduct {
	return SellerProduct{
		ID: r.ID, Slug: r.Slug, Name: r.Name, Description: r.Description, Status: r.Status,
		CategoryID: r.CategoryID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (r *Repository) Create(ctx context.Context, storeID, categoryID uuid.UUID, name, slug, description string) (SellerProduct, error) {
	row, err := r.q.CreateProduct(ctx, queries.CreateProductParams{
		StoreID: storeID, CategoryID: categoryID, Name: name, Slug: slug, Description: description,
	})
	if err != nil {
		return SellerProduct{}, translate(err, "create product")
	}
	return toSellerProduct(queries.GetStoreProductRow(row)), nil
}

// GetOwned returns the product only if it belongs to storeID.
func (r *Repository) GetOwned(ctx context.Context, storeID, productID uuid.UUID) (SellerProduct, error) {
	row, err := r.q.GetStoreProduct(ctx, queries.GetStoreProductParams{ID: productID, StoreID: storeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return SellerProduct{}, ErrNotFound
	}
	if err != nil {
		return SellerProduct{}, fmt.Errorf("get store product: %w", err)
	}
	return toSellerProduct(row), nil
}

func (r *Repository) ListOwned(ctx context.Context, storeID uuid.UUID) ([]SellerSummary, error) {
	rows, err := r.q.ListStoreProducts(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("list store products: %w", err)
	}
	out := make([]SellerSummary, len(rows))
	for i, row := range rows {
		out[i] = SellerSummary{
			ID: row.ID, Slug: row.Slug, Name: row.Name, Status: row.Status, CategoryID: row.CategoryID,
			VariantCount: row.VariantCount, TotalStock: row.TotalStock,
			LowestPrice: row.MinPrice, HighestPrice: row.MaxPrice, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *Repository) UpdateOwned(ctx context.Context, storeID uuid.UUID, p SellerProduct) (SellerProduct, error) {
	row, err := r.q.UpdateStoreProduct(ctx, queries.UpdateStoreProductParams{
		ID: p.ID, StoreID: storeID, Name: p.Name, Description: p.Description, CategoryID: p.CategoryID, Status: p.Status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SellerProduct{}, ErrNotFound
	}
	if err != nil {
		return SellerProduct{}, translate(err, "update product")
	}
	return toSellerProduct(queries.GetStoreProductRow(row)), nil
}

func (r *Repository) ReplaceImages(ctx context.Context, productID uuid.UUID, images []Image) error {
	if err := r.q.DeleteProductImages(ctx, productID); err != nil {
		return fmt.Errorf("delete images: %w", err)
	}
	for i, img := range images {
		if err := r.q.CreateProductImage(ctx, queries.CreateProductImageParams{
			ProductID: productID, Url: img.URL, Alt: img.Alt, SortOrder: int32(i),
		}); err != nil {
			return fmt.Errorf("create image: %w", err)
		}
	}
	return nil
}

func (r *Repository) CreateVariant(ctx context.Context, productID uuid.UUID, sku, name string, price decimal.Decimal, sortOrder int32) (Variant, error) {
	row, err := r.q.CreateVariant(ctx, queries.CreateVariantParams{
		ProductID: productID, Sku: sku, Name: name, Price: price, SortOrder: sortOrder,
	})
	if err != nil {
		return Variant{}, translate(err, "create variant")
	}
	return Variant{ID: row.ID, SKU: row.Sku, Name: row.Name, Price: row.Price}, nil
}

func (r *Repository) NextVariantSortOrder(ctx context.Context, productID uuid.UUID) (int32, error) {
	n, err := r.q.NextVariantSortOrder(ctx, productID)
	if err != nil {
		return 0, fmt.Errorf("next variant sort order: %w", err)
	}
	return n, nil
}

// GetOwnedVariant returns the variant only if it belongs to productID of storeID.
func (r *Repository) GetOwnedVariant(ctx context.Context, storeID, productID, variantID uuid.UUID) (Variant, error) {
	row, err := r.q.GetStoreVariant(ctx, queries.GetStoreVariantParams{VariantID: variantID, ProductID: productID, StoreID: storeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Variant{}, ErrVariantNotFound
	}
	if err != nil {
		return Variant{}, fmt.Errorf("get store variant: %w", err)
	}
	return Variant{ID: row.ID, SKU: row.Sku, Name: row.Name, Price: row.Price}, nil
}

func (r *Repository) UpdateVariant(ctx context.Context, v Variant) (Variant, error) {
	row, err := r.q.UpdateVariant(ctx, queries.UpdateVariantParams{ID: v.ID, Name: v.Name, Price: v.Price})
	if err != nil {
		return Variant{}, translate(err, "update variant")
	}
	return Variant{ID: row.ID, SKU: row.Sku, Name: row.Name, Price: row.Price}, nil
}

// SetSoldCount is for demo data only (cmd/seed-dev).
func (r *Repository) SetSoldCount(ctx context.Context, productID uuid.UUID, sold int32) error {
	return r.q.SetProductSoldCount(ctx, queries.SetProductSoldCountParams{ID: productID, SoldCount: sold})
}
