-- ============================================================================
-- Wishlist (practice feature). Every query filters by the caller's user_id,
-- so nobody can read or change another user's wishlist.
-- "Public product" = active product, active store, at least one variant
-- (the same rule as the catalog in products.sql).
-- ============================================================================

-- name: IsPublicProduct :one
SELECT EXISTS (
    SELECT 1
    FROM products p
    JOIN stores s ON s.id = p.store_id AND s.status = 'active'
    WHERE p.id = @id
      AND p.status = 'active'
      AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id)
)::boolean AS is_public;

-- name: AddWishlistItem :exec
-- Idempotent: adding a product twice, even by two requests at the same
-- moment, leaves exactly one row (the unique index decides, not the code).
INSERT INTO wishlist_items (user_id, product_id)
VALUES (@user_id, @product_id)
ON CONFLICT (user_id, product_id) DO NOTHING;

-- name: RemoveWishlistItem :exec
-- Idempotent as well: removing something that is not there is not an error.
DELETE FROM wishlist_items
WHERE user_id = @user_id AND product_id = @product_id;

-- name: ListWishlistItems :many
-- One page of the caller's wishlist, newest first. Products that stopped
-- being public (archived, store suspended) are left out but their row stays,
-- so they come back if the product is published again.
SELECT p.id AS product_id, p.slug, p.name, w.created_at AS added_at,
       s.slug AS store_slug, s.name AS store_name,
       v.min_price,
       -- LEFT JOIN: NULL when the product has no image; sqlc cannot infer
       -- nullability through LATERAL, so make it an explicit ''.
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       COUNT(*) OVER () AS total_count
FROM wishlist_items w
JOIN products p ON p.id = w.product_id AND p.status = 'active'
JOIN stores s ON s.id = p.store_id AND s.status = 'active'
JOIN LATERAL (
    SELECT MIN(pv.price)::numeric AS min_price
    FROM product_variants pv
    WHERE pv.product_id = p.id
    HAVING COUNT(*) > 0          -- no variant -> no row -> not public
) v ON true
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = p.id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE w.user_id = @user_id
ORDER BY w.created_at DESC, w.id
LIMIT @page_limit OFFSET @page_offset;

-- name: ListWishlistProductIDs :many
-- Ids of the caller's wishlisted products that are public, for the heart
-- buttons. Uses the (user_id, product_id) unique index.
SELECT w.product_id
FROM wishlist_items w
JOIN products p ON p.id = w.product_id AND p.status = 'active'
JOIN stores s ON s.id = p.store_id AND s.status = 'active'
WHERE w.user_id = @user_id
  AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id)
ORDER BY w.created_at DESC;
