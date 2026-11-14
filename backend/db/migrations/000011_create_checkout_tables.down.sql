-- Children first (foreign keys).
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_status_histories;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS checkouts;

-- Back to the 000008 list. NOT VALID keeps existing 'restock' rows (if any)
-- instead of deleting them: the ledger is history. Rolling back also does not
-- undo reservations already made (inventory.reserved_quantity); this down
-- migration is meant for development databases.
ALTER TABLE inventory_movements DROP CONSTRAINT inventory_movements_type_check;
ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_type_check
    CHECK (movement_type IN ('adjustment', 'reservation', 'release', 'consumption')) NOT VALID;
