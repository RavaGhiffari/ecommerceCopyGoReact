-- EXPLAIN (ANALYZE, BUFFERS) for the hot queries (Phase 7c). Read-only.
--
-- Run with `make perf-explain` against marketplace_perf (see scripts/seed-load.sql).
-- Every statement is PREPAREd with the same text as backend/db/queries/*.sql
-- (sqlc parameters become $1..$n, in the order of their first appearance), then
-- EXPLAINed through EXECUTE. That is how pgx runs them, so the planner sees
-- parameters, not literals.
--
-- PERF_GENERIC=1 forces generic plans (what pgx may switch to after five
-- executions); the default forces custom plans.

\set ON_ERROR_STOP on
\pset pager off

-- ---- sample parameters -----------------------------------------------------
SELECT md5('store-5')::uuid   AS store_id,           -- a seller with ~500 orders
       md5('user-205')::uuid  AS buyer_id,
       md5('product-5')::uuid AS product_id,
       (SELECT array_agg(id)::text FROM categories WHERE parent_id = md5('cat-0')::uuid) AS cat_tree,
       md5('cat-12')::uuid::text AS one_cat,
       (SELECT now() - interval '30 days') AS since
\gset

-- ---- catalog ---------------------------------------------------------------
PREPARE list_public(uuid[], uuid, uuid, text, text, int, int) AS
SELECT p.id, p.slug, p.name, p.category_id, p.sold_count, p.created_at,
       p.rating_count, p.rating_total,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location,
       v.min_price, v.max_price, v.in_stock,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt
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
  AND ($1::uuid[] IS NULL OR p.category_id = ANY ($1::uuid[]))
  AND ($2::uuid IS NULL OR p.store_id = $2::uuid)
  AND ($3::uuid IS NULL OR p.id <> $3::uuid)
  AND ($4::text IS NULL OR p.search @@ to_tsquery('simple', $4::text))
ORDER BY
  CASE WHEN $5::text = 'popular'    THEN p.sold_count END DESC,
  CASE WHEN $5::text = 'rating'     THEN p.rating_total::numeric / NULLIF(p.rating_count, 0) END DESC NULLS LAST,
  CASE WHEN $5::text = 'rating'     THEN p.rating_count END DESC,
  p.created_at DESC,
  p.id
LIMIT $6 OFFSET $7;

PREPARE list_by_price(uuid[], uuid, uuid, text, boolean, int, int) AS
WITH page AS (
    SELECT p.id, v.min_price, v.max_price, v.in_stock
    FROM products p
    JOIN stores s ON s.id = p.store_id AND s.status = 'active'
    JOIN (
        SELECT pv.product_id,
               MIN(pv.price)::numeric AS min_price,
               MAX(pv.price)::numeric AS max_price,
               COALESCE(BOOL_OR(i.available_quantity > 0), false)::boolean AS in_stock
        FROM product_variants pv
        LEFT JOIN inventory i ON i.variant_id = pv.id
        GROUP BY pv.product_id
    ) v ON v.product_id = p.id
    WHERE p.status = 'active'
      AND ($1::uuid[] IS NULL OR p.category_id = ANY ($1::uuid[]))
      AND ($2::uuid IS NULL OR p.store_id = $2::uuid)
      AND ($3::uuid IS NULL OR p.id <> $3::uuid)
      AND ($4::text IS NULL OR p.search @@ to_tsquery('simple', $4::text))
    ORDER BY
      CASE WHEN $5::boolean THEN v.min_price END DESC,
      CASE WHEN NOT $5::boolean THEN v.min_price END ASC,
      p.created_at DESC,
      p.id
    LIMIT $6 OFFSET $7
)
SELECT p.id, p.slug, p.name, p.category_id, p.sold_count, p.created_at,
       p.rating_count, p.rating_total,
       s.id AS store_id, s.slug AS store_slug, s.name AS store_name, s.location AS store_location,
       page.min_price, page.max_price, page.in_stock,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt
FROM page
JOIN products p ON p.id = page.id
JOIN stores s ON s.id = p.store_id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt
    FROM product_images pi
    WHERE pi.product_id = p.id
    ORDER BY pi.sort_order, pi.created_at
    LIMIT 1
) img ON true
ORDER BY
  CASE WHEN $5::boolean THEN page.min_price END DESC,
  CASE WHEN NOT $5::boolean THEN page.min_price END ASC,
  p.created_at DESC,
  p.id;

PREPARE count_public(uuid[], uuid, uuid, text) AS
SELECT COUNT(*)
FROM products p
JOIN stores s ON s.id = p.store_id AND s.status = 'active'
WHERE p.status = 'active'
  AND EXISTS (SELECT 1 FROM product_variants pv WHERE pv.product_id = p.id)
  AND ($1::uuid[] IS NULL OR p.category_id = ANY ($1::uuid[]))
  AND ($2::uuid IS NULL OR p.store_id = $2::uuid)
  AND ($3::uuid IS NULL OR p.id <> $3::uuid)
  AND ($4::text IS NULL OR p.search @@ to_tsquery('simple', $4::text));

PREPARE cart_lines(uuid) AS
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
    SELECT pi.url, pi.alt FROM product_images pi
    WHERE pi.product_id = p.id ORDER BY pi.sort_order, pi.created_at LIMIT 1
) img ON true
WHERE c.user_id = $1
ORDER BY s.name, s.id, ci.created_at, ci.id;

-- ---- orders ----------------------------------------------------------------
PREPARE order_items_for(uuid[]) AS
SELECT oi.id, oi.order_id, oi.variant_id, oi.product_id, p.slug AS product_slug,
       oi.product_name, oi.variant_name, oi.sku, oi.unit_price, oi.quantity, oi.line_total,
       COALESCE(img.url, '')::text AS image_url,
       COALESCE(img.alt, '')::text AS image_alt,
       r.id AS review_id, r.rating AS review_rating, r.comment AS review_comment
FROM order_items oi
JOIN products p ON p.id = oi.product_id
LEFT JOIN reviews r ON r.order_item_id = oi.id
LEFT JOIN LATERAL (
    SELECT pi.url, pi.alt FROM product_images pi
    WHERE pi.product_id = oi.product_id ORDER BY pi.sort_order, pi.created_at LIMIT 1
) img ON true
WHERE oi.order_id = ANY($1::uuid[])
ORDER BY oi.order_id, oi.created_at, oi.id;

PREPARE buyer_orders(uuid, int, int) AS
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.buyer_id = $1
ORDER BY o.created_at DESC, o.id
LIMIT $2 OFFSET $3;

PREPARE seller_orders(uuid, text, int, int) AS
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE o.store_id = $1
  AND ($2::text IS NULL OR o.status = $2::text)
ORDER BY o.created_at DESC, o.id
LIMIT $3 OFFSET $4;

PREPARE seller_orders_count(uuid, text) AS
SELECT count(*) FROM orders o
WHERE o.store_id = $1 AND ($2::text IS NULL OR o.status = $2::text);

PREPARE admin_orders(text, int, int) AS
SELECT o.id, o.checkout_id, o.order_number, o.status, o.shipping_method, o.shipping_fee,
       o.items_subtotal, o.total, o.store_id, s.slug AS store_slug, o.store_name,
       o.ship_recipient_name, o.ship_phone, o.ship_address_line, o.ship_city, o.ship_province,
       o.ship_postal_code, o.created_at, o.updated_at
FROM orders o
JOIN stores s ON s.id = o.store_id
WHERE ($1::text IS NULL OR o.status = $1::text)
ORDER BY o.created_at DESC, o.id
LIMIT $2 OFFSET $3;

PREPARE admin_orders_count(text) AS
SELECT count(*) FROM orders o WHERE ($1::text IS NULL OR o.status = $1::text);

-- ---- dashboards ------------------------------------------------------------
PREPARE seller_status_counts(uuid) AS
SELECT o.status, count(*)::bigint AS order_count FROM orders o WHERE o.store_id = $1 GROUP BY o.status;

PREPARE seller_revenue(uuid, timestamptz, text[]) AS
SELECT COALESCE(SUM(o.items_subtotal), 0)::numeric                                       AS revenue_all_time,
       COALESCE(SUM(o.items_subtotal) FILTER (WHERE o.created_at >= $2), 0)::numeric      AS revenue_window,
       COALESCE(SUM(u.units), 0)::bigint                                                  AS units_all_time,
       COALESCE(SUM(u.units) FILTER (WHERE o.created_at >= $2), 0)::bigint                AS units_window
FROM orders o
CROSS JOIN LATERAL (
    SELECT SUM(oi.quantity) AS units FROM order_items oi WHERE oi.order_id = o.id
) u
WHERE o.store_id = $1 AND o.status = ANY($3::text[]);

PREPARE seller_top_products(uuid, text[], timestamptz, int) AS
SELECT oi.product_id, p.name, p.status,
       SUM(oi.quantity)::bigint    AS units,
       SUM(oi.line_total)::numeric AS revenue
FROM order_items oi
JOIN orders o   ON o.id = oi.order_id
JOIN products p ON p.id = oi.product_id
WHERE o.store_id = $1 AND o.status = ANY($2::text[]) AND o.created_at >= $3
GROUP BY oi.product_id, p.name, p.status
ORDER BY units DESC, revenue DESC, p.name, oi.product_id
LIMIT $4;

PREPARE store_rating(uuid) AS
SELECT COALESCE(SUM(rating_count), 0)::int AS rating_count,
       COALESCE(SUM(rating_total), 0)::int AS rating_total
FROM products WHERE store_id = $1;

PREPARE admin_public_products AS
SELECT count(*)::bigint FROM products p JOIN stores s ON s.id = p.store_id
WHERE p.status = 'active' AND s.status = 'active'
  AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id);

PREPARE admin_gmv(timestamptz, text[]) AS
SELECT COALESCE(SUM(items_subtotal), 0)::numeric                                      AS gmv_all_time,
       COALESCE(SUM(items_subtotal) FILTER (WHERE created_at >= $1), 0)::numeric        AS gmv_window
FROM orders WHERE status = ANY($2::text[]);

-- ---- reviews ---------------------------------------------------------------
PREPARE review_distribution(uuid) AS
SELECT rating, COUNT(*)::int AS count FROM reviews WHERE product_id = $1 AND status = 'visible' GROUP BY rating;

PREPARE public_reviews(uuid, int, int) AS
SELECT r.id, r.rating, r.comment, r.created_at, r.updated_at,
       u.name AS reviewer_name, oi.variant_name, COUNT(*) OVER () AS total_count
FROM reviews r
JOIN users u ON u.id = r.user_id
JOIN order_items oi ON oi.id = r.order_item_id
WHERE r.product_id = $1 AND r.status = 'visible'
ORDER BY r.created_at DESC, r.id
LIMIT $2 OFFSET $3;

PREPARE pending_reviews(uuid, int, int) AS
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
    SELECT pi.url, pi.alt FROM product_images pi
    WHERE pi.product_id = oi.product_id ORDER BY pi.sort_order, pi.created_at LIMIT 1
) img ON true
WHERE o.buyer_id = $1 AND o.status = 'delivered' AND r.id IS NULL
ORDER BY o.updated_at DESC, oi.id
LIMIT $2 OFFSET $3;

PREPARE admin_reviews(text, int, int) AS
SELECT r.id, r.rating, r.comment, r.status, r.created_at, r.updated_at,
       r.product_id, oi.product_name, s.name AS store_name,
       u.name AS reviewer_name, u.email AS reviewer_email
FROM reviews r
JOIN users u ON u.id = r.user_id
JOIN stores s ON s.id = r.store_id
JOIN order_items oi ON oi.id = r.order_item_id
WHERE ($1::text IS NULL OR r.status = $1::text)
ORDER BY r.created_at DESC, r.id
LIMIT $2 OFFSET $3;

-- ---- run -------------------------------------------------------------------
SELECT CASE WHEN :'generic' = '1' THEN 'force_generic_plan' ELSE 'force_custom_plan' END AS mode \gset
SET plan_cache_mode = :mode;
\echo plan_cache_mode = :mode

\echo
\echo '#### ListPublicProducts: newest, no filter'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(NULL, NULL, NULL, NULL, 'newest', 24, 0);
\echo '#### ListPublicProducts: popular'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(NULL, NULL, NULL, NULL, 'popular', 24, 0);
\echo '#### ListPublicProducts: rating'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(NULL, NULL, NULL, NULL, 'rating', 24, 0);
\echo '#### ListPublicProducts: category tree, newest'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(:'cat_tree'::uuid[], NULL, NULL, NULL, 'newest', 24, 0);
\echo '#### ListPublicProducts: single category, popular'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(ARRAY[:'one_cat'::uuid], NULL, NULL, NULL, 'popular', 24, 0);
\echo '#### ListPublicProducts: search "kemeja:* & premium:*", newest'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(NULL, NULL, NULL, 'kemeja:* & premium:*', 'newest', 24, 0);
\echo '#### ListPublicProducts: store page'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(NULL, :'store_id', NULL, NULL, 'newest', 24, 0);
\echo '#### ListPublicProducts: related (category, exclude self)'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_public(ARRAY[:'one_cat'::uuid], NULL, :'product_id', NULL, 'popular', 8, 0);
\echo '#### ListPublicProductsByPrice: asc'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_by_price(NULL, NULL, NULL, NULL, false, 24, 0);
\echo '#### ListPublicProductsByPrice: desc, category tree'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_by_price(:'cat_tree'::uuid[], NULL, NULL, NULL, true, 24, 0);
\echo '#### ListPublicProductsByPrice: asc, search'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE list_by_price(NULL, NULL, NULL, 'kemeja:* & premium:*', false, 24, 0);
\echo '#### CountPublicProducts: no filter'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE count_public(NULL, NULL, NULL, NULL);
\echo '#### CountPublicProducts: category tree'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE count_public(:'cat_tree'::uuid[], NULL, NULL, NULL);
\echo '#### CountPublicProducts: search'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE count_public(NULL, NULL, NULL, 'kemeja:* & premium:*');

\echo '#### ListCartLines'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE cart_lines(:'buyer_id');
\echo '#### ListOrderItems (10 orders of the buyer)'
SELECT array_agg(id)::text AS buyer_order_ids FROM (SELECT id FROM orders WHERE buyer_id = :'buyer_id' LIMIT 10) x \gset
EXPLAIN (ANALYZE, BUFFERS) EXECUTE order_items_for(:'buyer_order_ids'::uuid[]);

\echo '#### ListBuyerOrders'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE buyer_orders(:'buyer_id', 10, 0);
\echo '#### ListSellerOrders: no status'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_orders(:'store_id', NULL, 10, 0);
\echo '#### ListSellerOrders: status = paid'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_orders(:'store_id', 'paid', 10, 0);
\echo '#### CountSellerOrders: no status'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_orders_count(:'store_id', NULL);
\echo '#### CountSellerOrders: status = paid'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_orders_count(:'store_id', 'paid');
\echo '#### ListAdminOrders: no status'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_orders(NULL, 10, 0);
\echo '#### ListAdminOrders: status = paid'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_orders('paid', 10, 0);
\echo '#### CountAdminOrders: no status'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_orders_count(NULL);
\echo '#### CountAdminOrders: status = paid'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_orders_count('paid');

\echo '#### Seller stats: status counts'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_status_counts(:'store_id');
\echo '#### Seller stats: revenue'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_revenue(:'store_id', :'since', ARRAY['paid','processing','shipped','delivered']);
\echo '#### Seller stats: top products'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE seller_top_products(:'store_id', ARRAY['paid','processing','shipped','delivered'], :'since', 5);
\echo '#### Store rating'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE store_rating(:'store_id');
\echo '#### Admin stats: public product count'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_public_products;
\echo '#### Admin stats: GMV'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_gmv(:'since', ARRAY['paid','processing','shipped','delivered']);

\echo '#### Review distribution (a reviewed product)'
SELECT product_id AS rp FROM reviews GROUP BY product_id ORDER BY count(*) DESC LIMIT 1 \gset
EXPLAIN (ANALYZE, BUFFERS) EXECUTE review_distribution(:'rp');
\echo '#### Public reviews page'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE public_reviews(:'rp', 10, 0);
\echo '#### Pending reviews of a buyer'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE pending_reviews(:'buyer_id', 10, 0);
\echo '#### Admin reviews: no status'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_reviews(NULL, 10, 0);
\echo '#### Admin reviews: status = hidden'
EXPLAIN (ANALYZE, BUFFERS) EXECUTE admin_reviews('hidden', 10, 0);
