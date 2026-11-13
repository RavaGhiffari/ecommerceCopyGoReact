-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES (@email, @password_hash, @name)
RETURNING id, email, name, phone, status, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, name, phone, status, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserCredentialsByEmail :one
-- The ONLY query that reads password_hash. Used by login.
SELECT id, password_hash, status
FROM users
WHERE LOWER(email) = LOWER(@email::text);

-- name: UpdateUserProfile :one
UPDATE users
SET name = @name, phone = @phone
WHERE id = @id
RETURNING id, email, name, phone, status, created_at, updated_at;

-- name: AssignRoleByName :execrows
-- Returns 0 rows affected when the role name does not exist.
INSERT INTO user_roles (user_id, role_id)
SELECT @user_id::uuid, r.id
FROM roles r
WHERE r.name = @role_name::text;

-- name: EnsureRoleByName :exec
-- Like AssignRoleByName, but having the role already is fine.
INSERT INTO user_roles (user_id, role_id)
SELECT @user_id::uuid, r.id
FROM roles r
WHERE r.name = @role_name::text
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: ListUserRoleNames :many
SELECT r.name
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.user_id = @user_id
ORDER BY r.name;
