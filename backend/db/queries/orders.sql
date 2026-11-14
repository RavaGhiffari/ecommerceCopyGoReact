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
       COALESCE(img.alt, '')::text AS image_alt
FROM order_items oi
JOIN products p ON p.id = oi.product_id
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
