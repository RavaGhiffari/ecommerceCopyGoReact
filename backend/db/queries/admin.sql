-- Phase 6d: admin reads and status changes for users and stores, and the
-- platform stats. Every route behind these is RequireRole(admin); every write
-- runs in one transaction together with its audit_logs row.
--
-- Search: @q is a LIKE pattern built in Go ('%' + escaped text + '%'), or
-- NULL for "no search". Status filters are NULL for "any status".

-- ---- users -----------------------------------------------------------------------

-- name: CountAdminUsers :one
SELECT count(*)
FROM users u
WHERE (sqlc.narg('status')::text IS NULL OR u.status = sqlc.narg('status')::text)
  AND (sqlc.narg('q')::text IS NULL OR u.email ILIKE sqlc.narg('q')::text OR u.name ILIKE sqlc.narg('q')::text);

-- name: ListAdminUsers :many
-- Newest first. Roles as a sorted array, so the list needs no second query.
SELECT u.id, u.email, u.name, u.status, u.created_at,
       COALESCE((SELECT array_agg(r.name ORDER BY r.name)
                 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                 WHERE ur.user_id = u.id), '{}')::text[] AS roles
FROM users u
WHERE (sqlc.narg('status')::text IS NULL OR u.status = sqlc.narg('status')::text)
  AND (sqlc.narg('q')::text IS NULL OR u.email ILIKE sqlc.narg('q')::text OR u.name ILIKE sqlc.narg('q')::text)
ORDER BY u.created_at DESC, u.id
LIMIT @page_limit OFFSET @page_offset;

-- name: LockUserForAdmin :one
-- Locks the target user, so two admins changing the same user run one after
-- the other and each audit row describes a real change.
SELECT u.id, u.email, u.name, u.status,
       EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
               WHERE ur.user_id = u.id AND r.name = 'admin') AS is_admin
FROM users u
WHERE u.id = @id
FOR UPDATE OF u;

-- name: SetUserStatus :exec
UPDATE users SET status = @status WHERE id = @id;

-- name: DeleteUserSessions :execrows
-- On suspend every session ends now. The session lookup already refuses
-- suspended users; deleting also means reactivation does not revive them.
DELETE FROM user_sessions WHERE user_id = @user_id;

-- ---- stores ----------------------------------------------------------------------

-- name: CountAdminStores :one
SELECT count(*)
FROM stores s
JOIN users u ON u.id = s.owner_id
WHERE (sqlc.narg('status')::text IS NULL OR s.status = sqlc.narg('status')::text)
  AND (sqlc.narg('q')::text IS NULL OR s.name ILIKE sqlc.narg('q')::text OR s.slug ILIKE sqlc.narg('q')::text
       OR u.email ILIKE sqlc.narg('q')::text);

-- name: ListAdminStores :many
SELECT s.id, s.name, s.slug, s.location, s.status, s.created_at,
       u.id AS owner_id, u.name AS owner_name, u.email AS owner_email,
       (SELECT count(*) FROM products p WHERE p.store_id = s.id AND p.status = 'active')::int AS active_product_count
FROM stores s
JOIN users u ON u.id = s.owner_id
WHERE (sqlc.narg('status')::text IS NULL OR s.status = sqlc.narg('status')::text)
  AND (sqlc.narg('q')::text IS NULL OR s.name ILIKE sqlc.narg('q')::text OR s.slug ILIKE sqlc.narg('q')::text
       OR u.email ILIKE sqlc.narg('q')::text)
ORDER BY s.created_at DESC, s.id
LIMIT @page_limit OFFSET @page_offset;

-- name: LockStoreForAdmin :one
SELECT id, name, slug, status
FROM stores
WHERE id = @id
FOR UPDATE;

-- name: SetStoreStatus :exec
-- A suspended store disappears from the catalog and its products are flagged
-- in carts (both already check stores.status). Existing orders keep their
-- lifecycle (PLAN.md Q10): nothing here touches them.
UPDATE stores SET status = @status WHERE id = @id;

-- ---- platform stats ----------------------------------------------------------------

-- name: AdminCountUsersByStatus :many
SELECT status, count(*)::bigint AS n FROM users GROUP BY status;

-- name: AdminCountStoresByStatus :many
SELECT status, count(*)::bigint AS n FROM stores GROUP BY status;

-- name: AdminCountPublicProducts :one
-- Products buyers can see: the catalog visibility rule.
SELECT count(*)::bigint
FROM products p
JOIN stores s ON s.id = p.store_id
WHERE p.status = 'active' AND s.status = 'active'
  AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id);

-- name: AdminCountOrdersByStatus :many
SELECT status, count(*)::bigint AS n FROM orders GROUP BY status;

-- name: AdminGMV :one
-- Gross merchandise value: the seller revenue rule (6c) over every store.
SELECT COALESCE(SUM(items_subtotal), 0)::numeric                                    AS gmv_all_time,
       COALESCE(SUM(items_subtotal) FILTER (WHERE created_at >= @since), 0)::numeric AS gmv_window
FROM orders
WHERE status = ANY(@statuses::text[]);

-- name: GetAdminStore :one
-- One store with the same columns as ListAdminStores (the response after a
-- status change).
SELECT s.id, s.name, s.slug, s.location, s.status, s.created_at,
       u.id AS owner_id, u.name AS owner_name, u.email AS owner_email,
       (SELECT count(*) FROM products p WHERE p.store_id = s.id AND p.status = 'active')::int AS active_product_count
FROM stores s
JOIN users u ON u.id = s.owner_id
WHERE s.id = @id;
