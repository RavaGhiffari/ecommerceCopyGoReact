-- Accounts for every role (customer, seller, admin). Roles live in user_roles.
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    -- Argon2id PHC string ($argon2id$v=19$m=...,t=...,p=...$salt$hash).
    -- Never selected except by the login query.
    password_hash TEXT         NOT NULL,
    name          VARCHAR(150) NOT NULL,
    phone         VARCHAR(30),
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended')),
    -- The application lowercases emails; the database refuses anything else,
    -- so "Rava@x.com" and "rava@x.com" can never become two accounts.
    CONSTRAINT users_email_lowercase_check CHECK (email = LOWER(email)),
    CONSTRAINT users_name_not_blank_check CHECK (LENGTH(TRIM(name)) > 0)
);

-- Case-insensitive uniqueness; also the index used by the login lookup
-- (WHERE LOWER(email) = LOWER($1)).
CREATE UNIQUE INDEX users_email_lower_key ON users (LOWER(email));

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
