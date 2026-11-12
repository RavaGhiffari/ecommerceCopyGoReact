-- RBAC: a fixed set of roles, and a many-to-many link to users.
CREATE TABLE roles (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) NOT NULL UNIQUE
);

-- The roles are part of the schema, not user data: code refers to them by
-- name, so they are created by the migration that creates the table.
INSERT INTO roles (name) VALUES ('customer'), ('seller'), ('admin');

CREATE TABLE user_roles (
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- RESTRICT: a role still assigned to someone cannot be deleted by accident.
    role_id    UUID        NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, role_id) -- also prevents assigning a role twice
);

-- The primary key already serves "roles of a user"; this serves
-- "users with a role" (admin screens later).
CREATE INDEX user_roles_role_id_idx ON user_roles (role_id);
