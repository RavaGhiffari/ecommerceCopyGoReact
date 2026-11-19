-- Bulk load for query-plan measurement (Phase 7c). NOT demo data.
--
-- Runs only through `make seed-load`, which drops and recreates the separate
-- marketplace_perf database, migrates it and pipes this file into it.
--
-- Volume: 5 200 users (200 sellers + 5 000 buyers), 200 stores, 60 categories,
-- 50 000 products, 200 000 variants + inventory, ~75 000 images,
-- 100 000 checkouts/orders, 200 000 order items, ~17 000 reviews.
--
-- Every id is md5('<kind>-<n>')::uuid, so rows can be addressed by number
-- without lookup tables, and a run is fully deterministic.

\set ON_ERROR_STOP on
\timing on

-- Price of variant v (0..3) of product p. Used by both the variants and the
-- order items, so snapshots equal the catalog price.
CREATE FUNCTION pg_temp.perf_price(p int, v int) RETURNS numeric
LANGUAGE sql IMMUTABLE AS
$$ SELECT (((p * 7919 + v * 104729) % 19900) + 100)::numeric * 100 $$;

-- ---- users, stores, categories -------------------------------------------
INSERT INTO users (id, email, password_hash, name)
SELECT md5('user-' || n)::uuid, 'perf-' || n || '@load.test', 'x', 'Perf User ' || n
FROM generate_series(0, 5199) n;

INSERT INTO stores (id, owner_id, name, slug, location, status)
SELECT md5('store-' || n)::uuid, md5('user-' || n)::uuid, 'Toko ' || n, 'toko-' || n,
       (ARRAY['Jakarta','Bandung','Surabaya','Medan','Makassar'])[1 + n % 5],
       CASE WHEN n % 100 = 99 THEN 'suspended' ELSE 'active' END
FROM generate_series(0, 199) n;

INSERT INTO categories (id, name, slug)
SELECT md5('cat-' || n)::uuid, 'Kategori ' || n, 'kategori-' || n FROM generate_series(0, 9) n;
INSERT INTO categories (id, parent_id, name, slug)
SELECT md5('cat-' || (10 + n))::uuid, md5('cat-' || (n / 5))::uuid,
       'Sub ' || n, 'sub-' || n
FROM generate_series(0, 49) n;

-- ---- catalog -------------------------------------------------------------
INSERT INTO products (id, store_id, category_id, name, slug, description, status, sold_count, created_at)
SELECT md5('product-' || n)::uuid,
       md5('store-' || (n % 200))::uuid,
       md5('cat-' || (10 + n % 50))::uuid,
       (ARRAY['Kemeja','Sepatu','Tas','Jam Tangan','Topi','Celana','Jaket','Kaos'])[1 + n % 8]
           || ' ' || (ARRAY['Premium','Klasik','Sport','Casual'])[1 + (n / 8) % 4] || ' ' || n,
       'produk-' || n,
       'Deskripsi produk nomor ' || n || ' untuk pengukuran performa pencarian teks.',
       CASE WHEN n % 20 = 19 THEN 'archived' ELSE 'active' END,
       (n * 7919) % 5000,
       now() - make_interval(secs => (n * 631) % 63072000)
FROM generate_series(0, 49999) n;

INSERT INTO product_variants (id, product_id, sku, name, price, sort_order)
SELECT md5('variant-' || p || '-' || v)::uuid, md5('product-' || p)::uuid,
       'SKU-' || p || '-' || v, 'Varian ' || v, pg_temp.perf_price(p, v), v
FROM generate_series(0, 49999) p, generate_series(0, 3) v;

INSERT INTO inventory (variant_id, available_quantity)
SELECT md5('variant-' || p || '-' || v)::uuid, (p * 31 + v * 17) % 200
FROM generate_series(0, 49999) p, generate_series(0, 3) v;

INSERT INTO product_images (product_id, url, alt, sort_order)
SELECT md5('product-' || p)::uuid, 'https://img.load.test/' || p || '-' || i || '.jpg', 'Foto ' || p, i
FROM generate_series(0, 49999) p, generate_series(0, 1) i
WHERE i = 0 OR p % 2 = 0;

-- ---- orders --------------------------------------------------------------
-- Order i: store i % 200, buyer 200 + (i * 7919 % 5000), two items from that
-- store's products (product number = store + 200 * k).
CREATE TEMP TABLE perf_orders AS
SELECT i,
       i % 200                                   AS s,
       200 + (i * 7919) % 5000                   AS b,
       (i % 200) + 200 * ((i * 31) % 250)        AS p1,
       (i % 200) + 200 * ((i * 31 + 1 + i % 7) % 250) AS p2,
       i % 4                                     AS v1,
       (i / 4) % 4                               AS v2,
       1 + i % 3                                 AS q1,
       1 + (i / 3) % 3                           AS q2,
       (ARRAY['delivered','delivered','delivered','delivered','delivered',
              'paid','processing','shipped','pending_payment','cancelled'])[1 + (i * 13) % 10] AS status,
       now() - make_interval(secs => (i * 315) % 31536000) AS created_at
FROM generate_series(0, 99999) i;

INSERT INTO checkouts (id, user_id, idempotency_key, request_hash, status,
                       items_total, shipping_total, grand_total, expires_at, created_at)
SELECT md5('checkout-' || i)::uuid, md5('user-' || b)::uuid, md5('idem-' || i)::uuid,
       decode(md5('hash-' || i) || md5('hash2-' || i), 'hex'), 'paid',
       pg_temp.perf_price(p1, v1) * q1 + pg_temp.perf_price(p2, v2) * q2, 15000,
       pg_temp.perf_price(p1, v1) * q1 + pg_temp.perf_price(p2, v2) * q2 + 15000,
       created_at + interval '1 day', created_at
FROM perf_orders;

INSERT INTO orders (id, checkout_id, store_id, buyer_id, order_number, status, shipping_method,
                    shipping_fee, items_subtotal, total, store_name, ship_recipient_name, ship_phone,
                    ship_address_line, ship_city, ship_province, ship_postal_code, created_at, updated_at)
SELECT md5('order-' || i)::uuid, md5('checkout-' || i)::uuid, md5('store-' || s)::uuid, md5('user-' || b)::uuid,
       'PK-20260101-' || upper(lpad(to_hex(i), 6, '0')), status, 'regular',
       15000, pg_temp.perf_price(p1, v1) * q1 + pg_temp.perf_price(p2, v2) * q2,
       pg_temp.perf_price(p1, v1) * q1 + pg_temp.perf_price(p2, v2) * q2 + 15000,
       'Toko ' || s, 'Penerima ' || b, '081234567890', 'Jl. Contoh No. ' || b, 'Jakarta', 'DKI Jakarta', '10110',
       created_at, created_at + interval '2 days'
FROM perf_orders;

INSERT INTO order_items (id, order_id, variant_id, product_id, product_name, variant_name, sku,
                         unit_price, quantity, line_total, created_at)
SELECT md5('item-' || i || '-1')::uuid, md5('order-' || i)::uuid, md5('variant-' || p1 || '-' || v1)::uuid,
       md5('product-' || p1)::uuid, 'Produk ' || p1, 'Varian ' || v1, 'SKU-' || p1 || '-' || v1,
       pg_temp.perf_price(p1, v1), q1, pg_temp.perf_price(p1, v1) * q1, created_at
FROM perf_orders
UNION ALL
SELECT md5('item-' || i || '-2')::uuid, md5('order-' || i)::uuid, md5('variant-' || p2 || '-' || v2)::uuid,
       md5('product-' || p2)::uuid, 'Produk ' || p2, 'Varian ' || v2, 'SKU-' || p2 || '-' || v2,
       pg_temp.perf_price(p2, v2), q2, pg_temp.perf_price(p2, v2) * q2, created_at
FROM perf_orders;

-- ---- reviews: every third delivered order reviews its first item -----------
INSERT INTO reviews (order_item_id, product_id, store_id, user_id, rating, comment, created_at)
SELECT md5('item-' || i || '-1')::uuid, md5('product-' || p1)::uuid, md5('store-' || s)::uuid,
       md5('user-' || b)::uuid, 3 + i % 3, 'Ulasan ' || i, created_at + interval '3 days'
FROM perf_orders
WHERE status = 'delivered' AND i % 3 = 0;

-- Keep the denormalised rating columns equal to the reviews (CHECK + invariant).
UPDATE products p
SET rating_count = r.c, rating_total = r.t
FROM (SELECT product_id, COUNT(*)::int AS c, SUM(rating)::int AS t FROM reviews GROUP BY product_id) r
WHERE r.product_id = p.id;

ANALYZE;

SELECT 'users' AS "table", count(*) FROM users UNION ALL
SELECT 'stores', count(*) FROM stores UNION ALL
SELECT 'products', count(*) FROM products UNION ALL
SELECT 'product_variants', count(*) FROM product_variants UNION ALL
SELECT 'inventory', count(*) FROM inventory UNION ALL
SELECT 'product_images', count(*) FROM product_images UNION ALL
SELECT 'orders', count(*) FROM orders UNION ALL
SELECT 'order_items', count(*) FROM order_items UNION ALL
SELECT 'reviews', count(*) FROM reviews;
