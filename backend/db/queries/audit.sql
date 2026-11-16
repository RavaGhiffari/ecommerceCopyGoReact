-- Phase 6d: audit log. Append-only: there is deliberately no UPDATE or DELETE.

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, target_type, target_id, details)
VALUES (@actor_id, @action, @target_type, @target_id, @details);

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_logs;

-- name: ListAuditLogs :many
-- Newest first (audit_logs_created_idx), with the actor's name and email.
SELECT a.id, a.action, a.target_type, a.target_id, a.details, a.created_at,
       a.actor_id, u.name AS actor_name, u.email AS actor_email
FROM audit_logs a
JOIN users u ON u.id = a.actor_id
ORDER BY a.created_at DESC, a.id
LIMIT @page_limit OFFSET @page_offset;
