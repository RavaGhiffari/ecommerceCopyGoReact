-- Phase 6a: product reviews.
--
-- One review per eligible purchase: an order item of a delivered order
-- (order_item_id UNIQUE). Buying the same product twice gives two reviews.
-- product_id, store_id and user_id are copied from the order by the service,
-- never taken from the request, so lists filter without joining the orders.

-- Denormalised aggregate, updated in the same transaction as the review write
-- with an atomic "UPDATE ... SET rating_total = rating_total + delta".
-- Invariant: rating_count / rating_total = COUNT / SUM(rating) of the
-- product's VISIBLE reviews. The CHECK keeps every average within 1..5.
ALTER TABLE products
    ADD COLUMN rating_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN rating_total INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT products_rating_check CHECK (
        rating_count >= 0 AND rating_total BETWEEN rating_count AND rating_count * 5
    );

CREATE TABLE reviews (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT everywhere: reviews are history, like the orders they point at.
    order_item_id UUID         NOT NULL REFERENCES order_items (id) ON DELETE RESTRICT,
    product_id    UUID         NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    store_id      UUID         NOT NULL REFERENCES stores (id) ON DELETE RESTRICT,
    user_id       UUID         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    rating        SMALLINT     NOT NULL,
    comment       TEXT         NOT NULL DEFAULT '',
    -- hidden = removed by an admin (Phase 6d); out of the aggregate and the
    -- public list, still visible to its author.
    status        VARCHAR(20)  NOT NULL DEFAULT 'visible',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT reviews_order_item_key UNIQUE (order_item_id),
    CONSTRAINT reviews_rating_check CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT reviews_comment_length_check CHECK (char_length(comment) <= 1000),
    CONSTRAINT reviews_status_check CHECK (status IN ('visible', 'hidden'))
);

-- Public list of a product (newest first) and the rating distribution.
CREATE INDEX reviews_product_visible_created_idx ON reviews (product_id, created_at DESC) WHERE status = 'visible';
-- "My reviews".
CREATE INDEX reviews_user_created_idx ON reviews (user_id, created_at DESC);

CREATE TRIGGER reviews_set_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
