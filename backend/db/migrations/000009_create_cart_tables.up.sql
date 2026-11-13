-- One cart per user, created lazily on the first "add to cart".
--
-- A cart is NOT a reservation: adding an item never touches `inventory`.
-- Stock is checked when adding/updating (so the user gets early feedback) and
-- will be checked again, under lock, at checkout.
CREATE TABLE carts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- UNIQUE: exactly one cart per user (also the index for "my cart").
    -- CASCADE: deleting a user deletes their cart; it is not history.
    user_id    UUID        NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER carts_set_updated_at
    BEFORE UPDATE ON carts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


CREATE TABLE cart_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id    UUID        NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    -- RESTRICT, like everywhere else: variants are never deleted (products are archived).
    variant_id UUID        NOT NULL REFERENCES product_variants (id) ON DELETE RESTRICT,
    quantity   INTEGER     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 1..99 per line (approved decision Q2: min(stock, 99); the stock part
    -- changes over time, so only the fixed bound can be a constraint).
    CONSTRAINT cart_items_quantity_check CHECK (quantity BETWEEN 1 AND 99),
    -- A variant appears at most once per cart; adding again increases the quantity.
    -- The unique index also serves "lines of a cart" lookups (cart_id first).
    CONSTRAINT cart_items_cart_variant_key UNIQUE (cart_id, variant_id)
);

CREATE TRIGGER cart_items_set_updated_at
    BEFORE UPDATE ON cart_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
