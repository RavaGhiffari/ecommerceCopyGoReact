-- ============================================================================
-- Public catalog
-- ============================================================================

-- name: ListPublicProducts :many
-- One page of the public catalog. Every filter is optional: a NULL parameter
-- switches its condition off. total_count (a window function) is the number of
-- matching rows before LIMIT, so one query returns both the page and the total.
--
-- A product is public only when it is active, its store is active, and it has
-- at least one variant (the inner LATERAL join drops products without one).
SELECT p.id, p.slug, p.name, p.category_id, p.sold_count, p.created_at,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location,
       v.min_price, v.max_price, v.in_stock,
       -- LEFT JOIN: NULL when the product has no image. COALESCE makes that an
       -- explicit '' (sqlc cannot infer nullability through a LATERAL join).
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       COUNT(*) OVER () AS total_count
FROM products p
JOIN stores s ON s.id = p.store_id AND s.status = 'active'
JOIN LATERAL (
    SELECT MIN(pv.price)::numeric AS min_price,
           MAX(pv.price)::numeric AS max_price,
           COALESCE(BOOL_OR(i.available_quantity > 0), false)::boolean AS in_stock
    FROM product_variants pv
    LEFT JOIN inventory i ON i.variant_id = pv.id
    WHERE pv.product_id = p.id
    HAVING COUNT(*) > 0
) v ON true
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = p.id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE p.status = 'active'
  AND (sqlc.narg('category_ids')::uuid[] IS NULL OR p.category_id = ANY (sqlc.narg('category_ids')::uuid[]))
  AND (sqlc.narg('store_id')::uuid IS NULL OR p.store_id = sqlc.narg('store_id')::uuid)
  AND (sqlc.narg('exclude_id')::uuid IS NULL OR p.id <> sqlc.narg('exclude_id')::uuid)
  -- The tsquery is built in Go from sanitised words (see products/search.go).
  AND (sqlc.narg('search')::text IS NULL OR p.search @@ to_tsquery('simple', sqlc.narg('search')::text))
ORDER BY
  CASE WHEN @sort::text = 'price_asc'  THEN v.min_price END ASC,
  CASE WHEN @sort::text = 'price_desc' THEN v.min_price END DESC,
  CASE WHEN @sort::text = 'popular'    THEN p.sold_count END DESC,
  -- Ties (and sort=newest) fall back to newest first; id makes paging stable.
  p.created_at DESC,
  p.id
LIMIT @page_limit OFFSET @page_offset;

-- name: GetPublicProduct :one
-- Product detail by its store-scoped address /stores/{store}/products/{product}.
SELECT p.id, p.slug, p.name, p.description, p.category_id, p.sold_count, p.created_at,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location
FROM products p
JOIN stores s ON s.id = p.store_id
WHERE s.slug = @store_slug
  AND p.slug = @product_slug
  AND p.status = 'active'
  AND s.status = 'active';

-- name: GetPublicProductCategory :one
-- Used by "related products": the category of an active product.
SELECT p.category_id
FROM products p
JOIN stores s ON s.id = p.store_id
WHERE p.id = @id AND p.status = 'active' AND s.status = 'active';

-- name: ListVariantsWithStock :many
SELECT pv.id, pv.sku, pv.name, pv.price, pv.sort_order,
       COALESCE(i.available_quantity, 0)::int AS available_quantity
FROM product_variants pv
LEFT JOIN inventory i ON i.variant_id = pv.id
WHERE pv.product_id = @product_id
ORDER BY pv.sort_order, pv.created_at;

-- name: ListProductImages :many
SELECT url, alt
FROM product_images
WHERE product_id = @product_id
ORDER BY sort_order, created_at;

-- ============================================================================
-- Seller management. Every query that touches an existing product takes the
-- seller's store_id: a product of another store simply "does not exist".
-- ============================================================================

-- name: CreateProduct :one
INSERT INTO products (store_id, category_id, name, slug, description)
VALUES (@store_id, @category_id, @name, @slug, @description)
RETURNING id, store_id, category_id, name, slug, description, status, sold_count, created_at, updated_at;

-- name: GetStoreProduct :one
SELECT id, store_id, category_id, name, slug, description, status, sold_count, created_at, updated_at
FROM products
WHERE id = @id AND store_id = @store_id;

-- name: ListStoreProducts :many
-- The seller's own products, archived ones included, with a stock summary.
SELECT p.id, p.slug, p.name, p.status, p.category_id, p.updated_at,
       COUNT(pv.id)::int AS variant_count,
       COALESCE(SUM(i.available_quantity), 0)::int AS total_stock,
       COALESCE(MIN(pv.price), 0)::numeric AS min_price,
       COALESCE(MAX(pv.price), 0)::numeric AS max_price
FROM products p
LEFT JOIN product_variants pv ON pv.product_id = p.id
LEFT JOIN inventory i ON i.variant_id = pv.id
WHERE p.store_id = @store_id
GROUP BY p.id
ORDER BY p.updated_at DESC;

-- name: UpdateStoreProduct :one
UPDATE products
SET name = @name, description = @description, category_id = @category_id, status = @status
WHERE id = @id AND store_id = @store_id
RETURNING id, store_id, category_id, name, slug, description, status, sold_count, created_at, updated_at;

-- name: CreateVariant :one
INSERT INTO product_variants (product_id, sku, name, price, sort_order)
VALUES (@product_id, @sku, @name, @price, @sort_order)
RETURNING id, sku, name, price, sort_order;

-- name: GetStoreVariant :one
-- A variant, but only if it belongs to the given product of the given store.
SELECT pv.id, pv.product_id, pv.sku, pv.name, pv.price, pv.sort_order
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.id = @variant_id
  AND pv.product_id = @product_id
  AND p.store_id = @store_id;

-- name: UpdateVariant :one
UPDATE product_variants
SET name = @name, price = @price
WHERE id = @id
RETURNING id, sku, name, price, sort_order;

-- name: NextVariantSortOrder :one
SELECT COALESCE(MAX(sort_order) + 1, 0)::int FROM product_variants WHERE product_id = @product_id;

-- name: CreateProductImage :exec
INSERT INTO product_images (product_id, url, alt, sort_order)
VALUES (@product_id, @url, @alt, @sort_order);

-- name: DeleteProductImages :exec
DELETE FROM product_images WHERE product_id = @product_id;

-- name: SetProductSoldCount :exec
-- Demo data only (cmd/seed-dev). Real sales update this during checkout.
UPDATE products SET sold_count = @sold_count WHERE id = @id;
