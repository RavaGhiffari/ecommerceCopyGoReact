-- ============================================================================
-- Reviews (Phase 6a). Ownership: every buyer query filters by the caller's
-- user id, so another buyer's order item or review is "not found".
-- ============================================================================

-- name: GetReviewableItem :one
-- The order item a buyer wants to review, only if it is in one of THEIR
-- orders. The service then checks the order status (delivered). No lock: in
-- the transition table "delivered" is final, so it cannot change afterwards.
SELECT oi.id, oi.product_id, o.store_id, o.status AS order_status
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE oi.id = @order_item_id
  AND o.buyer_id = @buyer_id;

-- name: InsertReview :one
-- ON CONFLICT DO NOTHING instead of a unique-violation error: an error would
-- abort the whole transaction, while "no row returned" lets the service
-- answer 409 REVIEW_EXISTS cleanly (a concurrent duplicate waits on the
-- unique index and then also lands here).
INSERT INTO reviews (order_item_id, product_id, store_id, user_id, rating, comment)
VALUES (@order_item_id, @product_id, @store_id, @user_id, @rating, @comment)
ON CONFLICT (order_item_id) DO NOTHING
RETURNING id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at;

-- name: AdjustProductRating :exec
-- Atomic counter update. The UPDATE takes the product row lock, so two
-- concurrent reviews of the same product run one after the other and neither
-- increment is lost (a read-then-write in Go would lose one).
UPDATE products
SET rating_count = rating_count + @count_delta::int,
    rating_total = rating_total + @total_delta::int
WHERE id = @id;

-- name: LockOwnReview :one
-- Edit: lock the caller's review row so two edits of it serialise and each
-- applies its delta against the rating the other one left.
SELECT id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at
FROM reviews
WHERE id = @id AND user_id = @user_id
FOR UPDATE;

-- name: UpdateReview :one
UPDATE reviews
SET rating = @rating, comment = @comment
WHERE id = @id
RETURNING id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at;

-- name: GetPublicProductRating :one
-- The aggregate of a public product (same visibility rule as the catalog,
-- minus the variant check, which the product page already applies).
SELECT p.rating_count, p.rating_total
FROM products p
JOIN stores s ON s.id = p.store_id
WHERE p.id = @id AND p.status = 'active' AND s.status = 'active';

-- name: ProductRatingDistribution :many
-- How many visible reviews have 1, 2, ... 5 stars. Missing stars = 0.
SELECT rating, COUNT(*)::int AS count
FROM reviews
WHERE product_id = @product_id AND status = 'visible'
GROUP BY rating;

-- name: ListProductReviews :many
-- Public list. Returns the reviewer's NAME only (masked in Go); never the
-- email or user id. The variant name comes from the order item snapshot.
SELECT r.id, r.rating, r.comment, r.created_at, r.updated_at,
       u.name AS reviewer_name, oi.variant_name,
       COUNT(*) OVER () AS total_count
FROM reviews r
JOIN users u ON u.id = r.user_id
JOIN order_items oi ON oi.id = r.order_item_id
WHERE r.product_id = @product_id AND r.status = 'visible'
ORDER BY r.created_at DESC, r.id
LIMIT @page_limit OFFSET @page_offset;

-- name: ListMyReviews :many
-- The caller's reviews, hidden ones included (so they know).
SELECT r.id, r.order_item_id, r.product_id, r.rating, r.comment, r.status, r.created_at, r.updated_at,
       oi.product_name, oi.variant_name, o.id AS order_id, o.order_number,
       p.slug AS product_slug, s.slug AS store_slug,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       COUNT(*) OVER () AS total_count
FROM reviews r
JOIN order_items oi ON oi.id = r.order_item_id
JOIN orders o ON o.id = oi.order_id
JOIN products p ON p.id = r.product_id
JOIN stores s ON s.id = r.store_id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = r.product_id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE r.user_id = @user_id
ORDER BY r.created_at DESC, r.id
LIMIT @page_limit OFFSET @page_offset;

-- name: ListPendingReviews :many
-- Order items the caller may still review: in a delivered order of theirs,
-- and without a review (anti-join on the unique order_item_id).
SELECT oi.id AS order_item_id, oi.product_id, oi.product_name, oi.variant_name,
       o.id AS order_id, o.order_number, o.updated_at AS delivered_at,
       p.slug AS product_slug, s.slug AS store_slug, o.store_name,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       COUNT(*) OVER () AS total_count
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
JOIN products p ON p.id = oi.product_id
JOIN stores s ON s.id = o.store_id
LEFT JOIN reviews r ON r.order_item_id = oi.id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = oi.product_id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE o.buyer_id = @buyer_id
  AND o.status = 'delivered'
  AND r.id IS NULL
ORDER BY o.updated_at DESC, oi.id
LIMIT @page_limit OFFSET @page_offset;

-- ---- admin moderation (6d) -----------------------------------------------------

-- name: CountAdminReviews :one
SELECT count(*) FROM reviews r
WHERE (sqlc.narg('status')::text IS NULL OR r.status = sqlc.narg('status')::text);

-- name: ListAdminReviews :many
-- Every review, newest first; status NULL = both. The admin sees who wrote it.
SELECT r.id, r.rating, r.comment, r.status, r.created_at, r.updated_at,
       r.product_id, oi.product_name, s.name AS store_name,
       u.name AS reviewer_name, u.email AS reviewer_email
FROM reviews r
JOIN users u ON u.id = r.user_id
JOIN stores s ON s.id = r.store_id
JOIN order_items oi ON oi.id = r.order_item_id
WHERE (sqlc.narg('status')::text IS NULL OR r.status = sqlc.narg('status')::text)
ORDER BY r.created_at DESC, r.id
LIMIT @page_limit OFFSET @page_offset;

-- name: LockReview :one
-- Moderation locks the row, like an author edit does, so the two apply their
-- aggregate deltas one at a time.
SELECT id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at
FROM reviews
WHERE id = @id
FOR UPDATE;

-- name: SetReviewStatus :one
-- Leaves updated_at alone (trigger is UPDATE OF rating, comment since 000015).
UPDATE reviews
SET status = @status
WHERE id = @id
RETURNING id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at;
