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

-- name: DeleteExpiredSessionsForUser :exec
DELETE FROM user_sessions
WHERE user_id = @user_id
  AND expires_at <= NOW();
