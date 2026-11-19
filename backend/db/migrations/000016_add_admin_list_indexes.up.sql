-- Phase 7c: indexes proven by EXPLAIN (ANALYZE) on the load database
-- (make seed-load, 100 000 orders, 16 666 reviews). Plans before/after are in
-- docs/PROJECT_STATE.md §1q.
--
-- The admin lists have no selective filter: they show "the newest 10 of all
-- orders / reviews". Without an index PostgreSQL read every row and sorted
-- (top-N heapsort over a parallel seq scan); with an index that matches the
-- ORDER BY it reads the first page and stops.

-- ListAdminOrders without ?status=  (ORDER BY created_at DESC, id)
CREATE INDEX orders_created_idx ON orders (created_at DESC, id);

-- ListAdminOrders with ?status=
CREATE INDEX orders_status_created_idx ON orders (status, created_at DESC, id);

-- ListAdminReviews without ?status=  (ORDER BY created_at DESC, id)
CREATE INDEX reviews_created_idx ON reviews (created_at DESC, id);
