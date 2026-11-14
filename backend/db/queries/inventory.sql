-- name: CreateInventory :exec
-- Starts at zero; the initial stock is then set through the normal path so it
-- gets a ledger row like every other change.
INSERT INTO inventory (variant_id) VALUES (@variant_id);

-- name: LockInventory :one
-- FOR UPDATE: other transactions that want to change this variant's stock wait
-- until we commit. Two simultaneous "set stock" requests (or, later, a
-- checkout) can then never overwrite each other's change.
SELECT available_quantity, reserved_quantity
FROM inventory
WHERE variant_id = @variant_id
FOR UPDATE;

-- name: SetAvailableQuantity :exec
UPDATE inventory SET available_quantity = @available_quantity WHERE variant_id = @variant_id;

-- name: InsertInventoryMovement :exec
INSERT INTO inventory_movements
    (variant_id, movement_type, available_delta, reserved_delta, reason, reference_type, reference_id, actor_id)
VALUES
    (@variant_id, @movement_type, @available_delta, @reserved_delta, @reason, @reference_type, @reference_id, @actor_id);

-- name: LockInventoryRows :many
-- Locks the stock rows of several variants at once (checkout).
--
-- ORDER BY variant_id is the deadlock guard: every checkout takes its locks
-- in the same global order, so two checkouts that share variants can never
-- each hold one row while waiting for the other's.
SELECT variant_id, available_quantity, reserved_quantity
FROM inventory
WHERE variant_id = ANY(@variant_ids::uuid[])
ORDER BY variant_id
FOR UPDATE;

-- name: ReserveInventory :execrows
-- Moves q units from available to reserved. The WHERE clause makes it a
-- conditional update: 0 rows changed = not enough stock, and nothing happens.
-- Together with the CHECK (available_quantity >= 0) constraint this is the
-- last guard against overselling, even if a caller forgot to lock first.
UPDATE inventory
SET available_quantity = available_quantity - @quantity::int,
    reserved_quantity  = reserved_quantity + @quantity::int
WHERE variant_id = @variant_id
  AND available_quantity >= @quantity::int;

-- name: ReleaseInventory :execrows
-- Gives reserved units back to available (cancel, payment failure, expiry).
-- Conditional like ReserveInventory: 0 rows = the reservation is not there,
-- which would be a bug, and the caller rolls back.
UPDATE inventory
SET available_quantity = available_quantity + @quantity::int,
    reserved_quantity  = reserved_quantity - @quantity::int
WHERE variant_id = @variant_id
  AND reserved_quantity >= @quantity::int;

-- name: ConsumeInventory :execrows
-- Payment succeeded: the reserved units are sold and leave the stock for good.
UPDATE inventory
SET reserved_quantity = reserved_quantity - @quantity::int
WHERE variant_id = @variant_id
  AND reserved_quantity >= @quantity::int;

-- name: RestockInventory :execrows
-- A paid order was refunded: the sold units come back as available.
UPDATE inventory
SET available_quantity = available_quantity + @quantity::int
WHERE variant_id = @variant_id;
