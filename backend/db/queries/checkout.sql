-- name: GetCheckoutByIdempotencyKey :one
-- A previous checkout of this user with this Idempotency-Key, if any.
SELECT id, request_hash
FROM checkouts
WHERE user_id = @user_id AND idempotency_key = @idempotency_key;

-- name: ListCartVariantIDs :many
-- The variants in the user's cart, in lock order (see LockInventoryRows).
SELECT ci.variant_id
FROM carts c
JOIN cart_items ci ON ci.cart_id = c.id
WHERE c.user_id = @user_id
ORDER BY ci.variant_id;

-- name: ListCheckoutLines :many
-- The cart with everything checkout needs: CURRENT price and stock, statuses,
-- the store owner (a buyer may have opened a store after adding the item) and
-- the store location (Same Day rule). Inside the checkout transaction this
-- runs after the inventory rows are locked, so available_quantity cannot
-- change before commit.
SELECT ci.id, ci.quantity,
       pv.id AS variant_id, pv.name AS variant_name, pv.sku, pv.price,
       p.id AS product_id, p.slug AS product_slug, p.name AS product_name, p.status AS product_status,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location,
       s.status AS store_status, s.owner_id AS store_owner_id,
       COALESCE(i.available_quantity, 0)::int AS available_quantity,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt
FROM carts c
JOIN cart_items ci ON ci.cart_id = c.id
JOIN product_variants pv ON pv.id = ci.variant_id
JOIN products p ON p.id = pv.product_id
JOIN stores s ON s.id = p.store_id
LEFT JOIN inventory i ON i.variant_id = pv.id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = p.id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE c.user_id = @user_id
ORDER BY s.name, s.id, ci.created_at, ci.id;

-- name: CreateCheckout :one
-- expires_at uses the database clock, like every later expiry check.
INSERT INTO checkouts
    (user_id, idempotency_key, request_hash, items_total, shipping_total, grand_total, expires_at)
VALUES
    (@user_id, @idempotency_key, @request_hash, @items_total, @shipping_total, @grand_total,
     NOW() + make_interval(secs => @ttl_seconds::float8))
RETURNING id, expires_at;

-- name: CreatePayment :exec
INSERT INTO payments (checkout_id, reference, amount, expires_at)
VALUES (@checkout_id, @reference, @amount, @expires_at);

-- name: DeleteCartItemsByVariants :execrows
-- Removes the purchased lines (by variant, so only what was actually bought).
DELETE FROM cart_items
WHERE cart_id = @cart_id AND variant_id = ANY(@variant_ids::uuid[]);

-- name: GetCheckout :one
-- Filtered by user_id: someone else's checkout id finds nothing (404).
SELECT c.id, c.status, c.items_total, c.shipping_total, c.grand_total,
       c.expires_at, c.paid_at, c.cancelled_at, c.created_at,
       p.id AS payment_id, p.reference AS payment_reference, p.provider AS payment_provider,
       p.amount AS payment_amount, p.status AS payment_status,
       p.expires_at AS payment_expires_at, p.paid_at AS payment_paid_at
FROM checkouts c
JOIN payments p ON p.checkout_id = c.id
WHERE c.id = @id AND c.user_id = @user_id;
