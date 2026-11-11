-- name: Ping :one
-- Ping proves the database answers a real query. Used by GET /ready.
SELECT 1::int AS ok;
