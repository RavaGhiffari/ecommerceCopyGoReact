-- name: InsertOrder :one
-- ON CONFLICT ... DO NOTHING: when the random order number is already taken
-- no row comes back (pgx.ErrNoRows) and the caller retries with a new number.
-- A plain unique-violation error would abort the whole checkout transaction.
INSERT INTO orders
    (checkout_id, store_id, buyer_id, order_number, shipping_method, shipping_fee, items_subtotal, total,
     store_name, ship_recipient_name, ship_phone, ship_address_line, ship_city, ship_province, ship_postal_code)
VALUES
    (@checkout_id, @store_id, @buyer_id, @order_number, @shipping_method, @shipping_fee, @items_subtotal, @total,
     @store_name, @ship_recipient_name, @ship_phone, @ship_address_line, @ship_city, @ship_province, @ship_postal_code)
ON CONFLICT (order_number) DO NOTHING
RETURNING id;

-- name: InsertOrderItem :exec
INSERT INTO order_items
    (order_id, variant_id, product_id, product_name, variant_name, sku, unit_price, quantity, line_total)
VALUES
    (@order_id, @variant_id, @product_id, @product_name, @variant_name, @sku, @unit_price, @quantity, @line_total);

-- name: InsertOrderStatusHistory :exec
INSERT INTO order_status_histories (order_id, from_status, to_status, actor_id, note)
VALUES (@order_id, @from_status, @to_status, @actor_id, @note);

-- name: ListOrdersByCheckout :many
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.checkout_id = @checkout_id
ORDER BY o.store_name, o.id;

-- name: CountBuyerOrders :one
SELECT count(*) FROM orders WHERE buyer_id = @buyer_id;

-- name: ListBuyerOrders :many
-- Newest first; served by orders_buyer_created_idx.
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.buyer_id = @buyer_id
ORDER BY o.created_at DESC, o.id
LIMIT @page_limit OFFSET @page_offset;

-- name: GetBuyerOrder :one
-- Filtered by buyer_id: another buyer's order id finds nothing (404).
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.id = @id AND o.buyer_id = @buyer_id;

-- name: ListOrderItems :many
-- Items of several orders in one query (no N+1 when listing a page of
-- orders). Names and prices are the snapshots; the product slug and image
-- are current and only used for links and thumbnails.
SELECT oi.id, oi.order_id, oi.variant_id, oi.product_id, p.slug AS product_slug,
       oi.product_name, oi.variant_name, oi.sku, oi.unit_price, oi.quantity, oi.line_total,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       -- The buyer's review of this item, if any (6a; at most one per item).
       r.id AS review_id, r.rating AS review_rating, r.comment AS review_comment
FROM order_items oi
JOIN products p ON p.id = oi.product_id
LEFT JOIN reviews r ON r.order_item_id = oi.id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = oi.product_id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
WHERE oi.order_id = ANY(@order_ids::uuid[])
ORDER BY oi.order_id, oi.created_at, oi.id;

-- name: ListOrderHistory :many
SELECT from_status, to_status, note, created_at
FROM order_status_histories
WHERE order_id = @order_id
ORDER BY created_at, id;

-- ---- status changes (5c) --------------------------------------------------------

-- name: LockCheckoutOrders :many
-- ORDER BY id: a fixed lock order, like the inventory rows.
SELECT id, checkout_id, status
FROM orders
WHERE checkout_id = @checkout_id
ORDER BY id
FOR UPDATE;

-- name: LockBuyerOrder :one
SELECT id, checkout_id, status
FROM orders
WHERE id = @id AND buyer_id = @buyer_id
FOR UPDATE;

-- name: SetOrderStatus :execrows
-- "AND status = @from_status": the update only happens from the status the
-- caller checked against the transition table.
UPDATE orders SET status = @to_status
WHERE id = @id AND status = @from_status;

-- name: ListCheckoutStock :many
-- What a checkout reserved, per variant, in lock order (variant_id).
SELECT oi.variant_id, oi.product_id, SUM(oi.quantity)::int AS quantity
FROM orders o
JOIN order_items oi ON oi.order_id = o.id
WHERE o.checkout_id = @checkout_id
GROUP BY oi.variant_id, oi.product_id
ORDER BY oi.variant_id;

-- name: ListOrderStock :many
SELECT variant_id, product_id, quantity
FROM order_items
WHERE order_id = @order_id
ORDER BY variant_id;

-- name: AddProductSoldCount :exec
-- Denormalised counter for "popular" sorting: + on payment, - on refund.
-- CHECK (sold_count >= 0) guards against going below zero.
UPDATE products SET sold_count = sold_count + @delta::int WHERE id = @id;

-- ---- seller (5d) -----------------------------------------------------------------
-- Every seller query filters by store_id, which the service takes from the
-- session (the caller's own store). Another store's order id finds nothing.
-- status is optional: NULL lists every status.

-- name: CountStoreOrders :one
SELECT count(*)
FROM orders o
WHERE o.store_id = @store_id
  AND (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text);

-- name: ListStoreOrders :many
-- Newest first; served by orders_store_status_created_idx.
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.store_id = @store_id
  AND (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
ORDER BY o.created_at DESC, o.id
LIMIT @page_limit OFFSET @page_offset;

-- name: GetStoreOrder :one
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.id = @id AND o.store_id = @store_id;

-- name: LockStoreOrder :one
-- The same row lock as LockBuyerOrder: "seller starts processing" and
-- "buyer cancels" for one order run one after the other.
SELECT id, checkout_id, status
FROM orders
WHERE id = @id AND store_id = @store_id
FOR UPDATE;

-- ---- seller stats (6c) -----------------------------------------------------------
-- GET /seller/stats. Every query filters by the caller's store_id (from the
-- session). "Revenue" counts only orders whose money the store keeps:
-- @statuses is passed from Go (orders.RevenueStatuses = paid, processing,
-- shipped, delivered), so pending, cancelled and refunded orders never count.
-- The 30-day window is on orders.created_at: a checkout expires after 30
-- minutes, so an order is paid (or cancelled) within minutes of being created.

-- name: CountStoreOrdersByStatus :many
SELECT o.status, count(*)::bigint AS order_count
FROM orders o
WHERE o.store_id = @store_id
GROUP BY o.status;

-- name: StoreRevenueTotals :one
-- items_subtotal, not total: shipping fees are not the seller's sales.
SELECT COALESCE(SUM(o.items_subtotal), 0)::numeric                                      AS revenue_all_time,
       COALESCE(SUM(o.items_subtotal) FILTER (WHERE o.created_at >= @since), 0)::numeric AS revenue_window,
       COALESCE(SUM(u.units), 0)::bigint                                                AS units_all_time,
       COALESCE(SUM(u.units) FILTER (WHERE o.created_at >= @since), 0)::bigint          AS units_window
FROM orders o
CROSS JOIN LATERAL (
    SELECT SUM(oi.quantity) AS units FROM order_items oi WHERE oi.order_id = o.id
) u
WHERE o.store_id = @store_id
  AND o.status = ANY(@statuses::text[]);

-- name: ListStoreTopProducts :many
-- Best sellers of the window by units, then by revenue. The name is the
-- product's current one (the seller knows their products by it); archived
-- products still count, they were real sales.
SELECT oi.product_id, p.name, p.status,
       SUM(oi.quantity)::bigint    AS units,
       SUM(oi.line_total)::numeric AS revenue
FROM order_items oi
JOIN orders o   ON o.id = oi.order_id
JOIN products p ON p.id = oi.product_id
WHERE o.store_id = @store_id
  AND o.status = ANY(@statuses::text[])
  AND o.created_at >= @since
GROUP BY oi.product_id, p.name, p.status
ORDER BY units DESC, revenue DESC, p.name, oi.product_id
LIMIT @top_limit;

-- name: GetStoreRatingTotals :one
-- Same rule as the public store page: every product, archived included.
SELECT COALESCE(SUM(rating_count), 0)::int AS rating_count,
       COALESCE(SUM(rating_total), 0)::int AS rating_total
FROM products
WHERE store_id = @store_id;

-- ---- admin: platform orders (6d), read-only -----------------------------------------

-- name: CountAllOrders :one
SELECT count(*)
FROM orders o
WHERE (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text);

-- name: ListAllOrders :many
-- Every store, newest first. No index on created_at alone yet: fine at this
-- size, revisit with EXPLAIN in Phase 7.
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')::text)
ORDER BY o.created_at DESC, o.id
LIMIT @page_limit OFFSET @page_offset;

-- name: GetAnyOrder :one
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.id = @id;
