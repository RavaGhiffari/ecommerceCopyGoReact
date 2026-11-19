-- name: CreateSession :exec
INSERT INTO user_sessions (user_id, token_hash, expires_at)
VALUES (@user_id, @token_hash, @expires_at);

-- name: GetActiveSession :one
-- Runs on every authenticated request. Expired sessions and suspended users
-- are filtered here, so a suspended user is logged out immediately.
SELECT s.id AS session_id, s.user_id
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = @token_hash
  AND s.expires_at > NOW()
  AND u.status = 'active';

-- name: DeleteSessionByTokenHash :exec
DELETE FROM user_sessions
WHERE token_hash = @token_hash;

-- name: DeleteExpiredSessions :execrows
-- Background cleanup (auth.Service.CleanupExpiredSessions). Expired sessions
-- are already useless (GetActiveSession ignores them); this only stops the
-- table from growing with users who never log in again. The LIMIT keeps one
-- statement short; the caller repeats it until less than a batch is deleted.
-- "<= NOW()" is the exact complement of GetActiveSession's "> NOW()".
DELETE FROM user_sessions
WHERE id IN (
    SELECT id FROM user_sessions
    WHERE expires_at <= NOW()
    LIMIT @batch_size
);

-- name: DeleteExpiredSessionsForUser :exec
DELETE FROM user_sessions
WHERE user_id = @user_id
  AND expires_at <= NOW();
