-- name: EnsureCart :one
-- Returns the user's cart, creating it if needed.
--
-- Inside a transaction this also LOCKS the cart row until commit: a fresh
-- INSERT locks the new row, and the ON CONFLICT ... DO UPDATE branch locks the
-- existing one. All changes to one cart therefore run one after another, so
-- two simultaneous "add" requests cannot both read the old quantity.
INSERT INTO carts (user_id)
VALUES (@user_id)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id;

-- name: LockCartByUser :one
-- Like EnsureCart, but never creates a cart (used by update/remove).
SELECT id FROM carts WHERE user_id = @user_id FOR UPDATE;

-- name: GetVariantForCart :one
-- Everything needed to decide whether a variant may go into a cart.
SELECT pv.id, pv.price,
       p.status AS product_status,
       s.status AS store_status,
       s.owner_id AS store_owner_id,
       COALESCE(i.available_quantity, 0)::int AS available_quantity
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
JOIN stores s ON s.id = p.store_id
LEFT JOIN inventory i ON i.variant_id = pv.id
WHERE pv.id = @variant_id;

-- name: GetCartItemQuantity :one
SELECT quantity FROM cart_items WHERE cart_id = @cart_id AND variant_id = @variant_id;

-- name: UpsertCartItem :exec
-- The service computes the final quantity while holding the cart lock.
INSERT INTO cart_items (cart_id, variant_id, quantity)
VALUES (@cart_id, @variant_id, @quantity)
ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = EXCLUDED.quantity;

-- name: GetCartItem :one
-- Only finds the item if it is in the given cart (ownership check).
SELECT id, variant_id, quantity
FROM cart_items
WHERE id = @id AND cart_id = @cart_id;

-- name: SetCartItemQuantity :exec
UPDATE cart_items SET quantity = @quantity WHERE id = @id AND cart_id = @cart_id;

-- name: DeleteCartItem :execrows
-- Joins through carts so another user's item id matches nothing (0 rows -> 404).
DELETE FROM cart_items ci
USING carts c
WHERE ci.id = @id
  AND ci.cart_id = c.id
  AND c.user_id = @user_id;

-- name: ListCartLines :many
-- The whole cart with CURRENT prices and stock. No visibility filter here on
-- purpose: archived products or suspended stores stay in the cart and are
-- flagged by the service, instead of silently disappearing.
SELECT ci.id, ci.quantity, ci.created_at,
       pv.id AS variant_id, pv.name AS variant_name, pv.sku, pv.price,
       p.id AS product_id, p.slug AS product_slug, p.name AS product_name, p.status AS product_status,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location,
       s.status AS store_status,
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
