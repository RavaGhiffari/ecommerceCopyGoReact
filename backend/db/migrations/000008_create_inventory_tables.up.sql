-- Stock per variant. One row per variant, so variant_id is the primary key.
CREATE TABLE inventory (
    variant_id         UUID        PRIMARY KEY REFERENCES product_variants (id) ON DELETE RESTRICT,
    available_quantity INTEGER     NOT NULL DEFAULT 0,
    -- Held by unpaid checkouts (used from the checkout phase on).
    reserved_quantity  INTEGER     NOT NULL DEFAULT 0,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- The database itself refuses negative stock, whatever the code does.
    CONSTRAINT inventory_available_check CHECK (available_quantity >= 0),
    CONSTRAINT inventory_reserved_check CHECK (reserved_quantity >= 0)
);

CREATE TRIGGER inventory_set_updated_at
    BEFORE UPDATE ON inventory
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


-- Append-only ledger: every change to `inventory` writes one row here, in the
-- same transaction. Summing the deltas of a variant must always give its
-- current quantities, which makes stock auditable ("who changed it, and why?").
CREATE TABLE inventory_movements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id      UUID         NOT NULL REFERENCES product_variants (id) ON DELETE RESTRICT,
    movement_type   VARCHAR(20)  NOT NULL,
    available_delta INTEGER      NOT NULL DEFAULT 0,
    reserved_delta  INTEGER      NOT NULL DEFAULT 0,
    reason          VARCHAR(200) NOT NULL DEFAULT '',
    -- What caused it, e.g. ('order', <order id>) later. NULL for manual changes.
    reference_type  VARCHAR(30),
    reference_id    UUID,
    -- Who did it. NULL for system actions (e.g. an expired reservation).
    actor_id        UUID         REFERENCES users (id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT inventory_movements_type_check
        CHECK (movement_type IN ('adjustment', 'reservation', 'release', 'consumption')),
    CONSTRAINT inventory_movements_nonzero_check
        CHECK (available_delta <> 0 OR reserved_delta <> 0),
    CONSTRAINT inventory_movements_reference_check
        CHECK ((reference_type IS NULL) = (reference_id IS NULL))
);

CREATE INDEX inventory_movements_variant_created_idx ON inventory_movements (variant_id, created_at DESC);
