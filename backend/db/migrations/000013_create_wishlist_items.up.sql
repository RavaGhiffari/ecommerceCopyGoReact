-- Wishlist (practice feature, not part of the phase plan).
--
-- One row = one product saved by one user. Rows are only inserted or
-- deleted, never updated, so there is no updated_at.
CREATE TABLE wishlist_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- CASCADE: a wishlist is not history; it goes with its user.
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- RESTRICT: products are archived, never deleted (same as orders/reviews).
    product_id UUID        NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One product at most once per user. The database enforces it, so two
    -- concurrent "add" requests cannot create a duplicate; the API uses
    -- INSERT ... ON CONFLICT (user_id, product_id) DO NOTHING to stay idempotent.
    -- user_id first: its index also serves "is this product in MY wishlist"
    -- and "remove this product from MY wishlist" (WHERE user_id = ? AND product_id = ?).
    CONSTRAINT wishlist_items_user_product_key UNIQUE (user_id, product_id)
);

-- "My wishlist, newest first" (GET /me/wishlist).
CREATE INDEX wishlist_items_user_created_idx ON wishlist_items (user_id, created_at DESC);
