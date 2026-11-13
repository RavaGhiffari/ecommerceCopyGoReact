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
