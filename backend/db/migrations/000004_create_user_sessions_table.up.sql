-- Server-side sessions. The browser holds a random token in an HttpOnly
-- cookie; we store only SHA-256(token). If this table leaks, the hashes
-- cannot be turned back into working cookies.
CREATE TABLE user_sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash BYTEA       NOT NULL UNIQUE, -- UNIQUE also indexes the per-request lookup
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_sessions_token_hash_length_check CHECK (OCTET_LENGTH(token_hash) = 32),
    CONSTRAINT user_sessions_expiry_check CHECK (expires_at > created_at)
);

-- "All sessions of a user": cleanup on login, and logging a user out
-- everywhere when an admin suspends them (later phase).
CREATE INDEX user_sessions_user_id_idx ON user_sessions (user_id);
