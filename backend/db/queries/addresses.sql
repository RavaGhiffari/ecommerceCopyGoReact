-- Every query filters by user_id: an address id that belongs to someone else
-- behaves exactly like an id that does not exist (404, never 403).

-- name: LockUserForAddressChange :one
-- Serialises address changes of ONE user (limit of 10, default switching).
-- FOR NO KEY UPDATE, not FOR UPDATE: it still blocks a second address change
-- for the same user, but does not block inserts that only reference the user
-- through a foreign key (e.g. a new login session).
SELECT id FROM users WHERE id = @user_id FOR NO KEY UPDATE;

-- name: CountUserAddresses :one
SELECT COUNT(*)::int FROM user_addresses WHERE user_id = @user_id;

-- name: ListUserAddresses :many
-- Default first, then newest first.
SELECT id, user_id, label, recipient_name, phone, address_line, city, province, postal_code,
       is_default, created_at, updated_at
FROM user_addresses
WHERE user_id = @user_id
ORDER BY is_default DESC, created_at DESC, id;

-- name: GetUserAddress :one
SELECT id, user_id, label, recipient_name, phone, address_line, city, province, postal_code,
       is_default, created_at, updated_at
FROM user_addresses
WHERE id = @id AND user_id = @user_id;

-- name: CreateUserAddress :one
INSERT INTO user_addresses (user_id, label, recipient_name, phone, address_line, city, province, postal_code, is_default)
VALUES (@user_id, @label, @recipient_name, @phone, @address_line, @city, @province, @postal_code, @is_default)
RETURNING id, user_id, label, recipient_name, phone, address_line, city, province, postal_code,
          is_default, created_at, updated_at;

-- name: UpdateUserAddress :one
UPDATE user_addresses
SET label = @label, recipient_name = @recipient_name, phone = @phone, address_line = @address_line,
    city = @city, province = @province, postal_code = @postal_code
WHERE id = @id AND user_id = @user_id
RETURNING id, user_id, label, recipient_name, phone, address_line, city, province, postal_code,
          is_default, created_at, updated_at;

-- name: DeleteUserAddress :one
-- Returns whether the deleted row was the default, so the service knows it
-- must promote another address.
DELETE FROM user_addresses
WHERE id = @id AND user_id = @user_id
RETURNING is_default;

-- name: ClearDefaultUserAddress :exec
-- Must run BEFORE a new default is set: the partial unique index allows only
-- one default row per user at any moment, even inside a transaction.
UPDATE user_addresses SET is_default = FALSE WHERE user_id = @user_id AND is_default;

-- name: SetDefaultUserAddress :one
UPDATE user_addresses SET is_default = TRUE
WHERE id = @id AND user_id = @user_id
RETURNING id, user_id, label, recipient_name, phone, address_line, city, province, postal_code,
          is_default, created_at, updated_at;

-- name: PromoteNewestUserAddress :exec
-- After the default was deleted, the newest remaining address becomes the
-- default (no-op when none is left).
UPDATE user_addresses SET is_default = TRUE
WHERE id = (
    SELECT a.id FROM user_addresses a
    WHERE a.user_id = @user_id
    ORDER BY a.created_at DESC, a.id
    LIMIT 1
);
