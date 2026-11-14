-- Phase 5b: checkout, per-store orders, their history, and the payment row.
--
-- Shape (approved plan, PLAN.md §5.2):
--   checkouts 1 ── n orders (one per store) 1 ── n order_items
--             1 ── 1 payments 1 ── n payment_events
--   orders    1 ── n order_status_histories
--
-- Everything here is HISTORY: rows are never deleted, so every foreign key is
-- ON DELETE RESTRICT. Products are archived, stores suspended and users
-- suspended instead of deleted, which keeps these references valid forever.
-- Money is NUMERIC(14,2): a total can be larger than one NUMERIC(12,2) price.

CREATE TABLE checkouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- Sent by the client in the Idempotency-Key header. The same key twice
    -- returns the first checkout instead of creating a second one.
    idempotency_key UUID          NOT NULL,
    -- SHA-256 of the canonical request, to detect a key reused for a
    -- different request (409 instead of silently returning the old one).
    request_hash    BYTEA         NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'pending_payment',
    items_total     NUMERIC(14,2) NOT NULL,
    shipping_total  NUMERIC(14,2) NOT NULL,
    grand_total     NUMERIC(14,2) NOT NULL,
    expires_at      TIMESTAMPTZ   NOT NULL,
    paid_at         TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT checkouts_user_idempotency_key UNIQUE (user_id, idempotency_key),
    CONSTRAINT checkouts_status_check
        CHECK (status IN ('pending_payment', 'paid', 'cancelled', 'expired')),
    CONSTRAINT checkouts_request_hash_check CHECK (octet_length(request_hash) = 32),
    CONSTRAINT checkouts_amounts_check
        CHECK (items_total >= 0 AND shipping_total >= 0 AND grand_total = items_total + shipping_total)
);

-- The expiry sweeper (5c) only ever looks at unpaid checkouts.
CREATE INDEX checkouts_pending_expires_idx ON checkouts (expires_at) WHERE status = 'pending_payment';

CREATE TRIGGER checkouts_set_updated_at
    BEFORE UPDATE ON checkouts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


CREATE TABLE orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_id         UUID          NOT NULL REFERENCES checkouts (id) ON DELETE RESTRICT,
    store_id            UUID          NOT NULL REFERENCES stores (id) ON DELETE RESTRICT,
    buyer_id            UUID          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- Human-readable, e.g. PK-20261114-7KQ2M9.
    order_number        VARCHAR(20)   NOT NULL,
    status              VARCHAR(20)   NOT NULL DEFAULT 'pending_payment',
    shipping_method     VARCHAR(20)   NOT NULL,
    shipping_fee        NUMERIC(14,2) NOT NULL,
    items_subtotal      NUMERIC(14,2) NOT NULL,
    total               NUMERIC(14,2) NOT NULL,
    -- Snapshots: editing or deleting the address, or renaming the store,
    -- must never change an order that already exists.
    store_name          VARCHAR(100)  NOT NULL,
    ship_recipient_name VARCHAR(100)  NOT NULL,
    ship_phone          VARCHAR(16)   NOT NULL,
    ship_address_line   VARCHAR(255)  NOT NULL,
    ship_city           VARCHAR(100)  NOT NULL,
    ship_province       VARCHAR(100)  NOT NULL,
    ship_postal_code    VARCHAR(5)    NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT orders_order_number_key UNIQUE (order_number),
    -- One order per store per checkout. Also serves "orders of a checkout".
    CONSTRAINT orders_checkout_store_key UNIQUE (checkout_id, store_id),
    -- Crockford base-32: digits and capitals without I, L, O, U.
    CONSTRAINT orders_order_number_format_check
        CHECK (order_number ~ '^PK-[0-9]{8}-[0-9A-HJKMNP-TV-Z]{6}$'),
    CONSTRAINT orders_status_check CHECK (status IN
        ('pending_payment', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded')),
    CONSTRAINT orders_shipping_method_check CHECK (shipping_method IN ('regular', 'express', 'same_day')),
    CONSTRAINT orders_amounts_check
        CHECK (shipping_fee >= 0 AND items_subtotal >= 0 AND total = items_subtotal + shipping_fee)
);

CREATE INDEX orders_buyer_created_idx ON orders (buyer_id, created_at DESC);               -- "my orders"
CREATE INDEX orders_store_status_created_idx ON orders (store_id, status, created_at DESC); -- seller list (5d)

CREATE TRIGGER orders_set_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


CREATE TABLE order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID          NOT NULL REFERENCES orders (id) ON DELETE RESTRICT,
    variant_id   UUID          NOT NULL REFERENCES product_variants (id) ON DELETE RESTRICT,
    product_id   UUID          NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    -- Snapshots taken at checkout; later price or name changes do not apply.
    product_name VARCHAR(200)  NOT NULL,
    variant_name VARCHAR(100)  NOT NULL,
    sku          VARCHAR(64)   NOT NULL,
    unit_price   NUMERIC(12,2) NOT NULL,
    quantity     INTEGER       NOT NULL,
    line_total   NUMERIC(14,2) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Also the index for "items of an order" (order_id first).
    CONSTRAINT order_items_order_variant_key UNIQUE (order_id, variant_id),
    CONSTRAINT order_items_quantity_check CHECK (quantity BETWEEN 1 AND 99),
    CONSTRAINT order_items_amounts_check CHECK (unit_price >= 0 AND line_total = unit_price * quantity)
);


-- Append-only: one row per status change, including the creation (from NULL).
CREATE TABLE order_status_histories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID         NOT NULL REFERENCES orders (id) ON DELETE RESTRICT,
    from_status VARCHAR(20),
    to_status   VARCHAR(20)  NOT NULL,
    -- Who did it; NULL for the system (payment result, expiry).
    actor_id    UUID         REFERENCES users (id) ON DELETE RESTRICT,
    note        VARCHAR(500) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT order_status_histories_status_check CHECK (
        (from_status IS NULL OR from_status IN
            ('pending_payment', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'))
        AND to_status IN
            ('pending_payment', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'))
);

CREATE INDEX order_status_histories_order_idx ON order_status_histories (order_id, created_at);


-- One simulated payment per checkout (decision Q4), covering all its orders.
CREATE TABLE payments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_id UUID          NOT NULL REFERENCES checkouts (id) ON DELETE RESTRICT,
    reference   VARCHAR(40)   NOT NULL,
    provider    VARCHAR(20)   NOT NULL DEFAULT 'simulated',
    amount      NUMERIC(14,2) NOT NULL,
    status      VARCHAR(20)   NOT NULL DEFAULT 'pending',
    expires_at  TIMESTAMPTZ   NOT NULL,
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT payments_checkout_key UNIQUE (checkout_id),
    CONSTRAINT payments_reference_key UNIQUE (reference),
    CONSTRAINT payments_provider_check CHECK (provider IN ('simulated')),
    CONSTRAINT payments_status_check
        CHECK (status IN ('pending', 'succeeded', 'failed', 'expired', 'cancelled')),
    CONSTRAINT payments_amount_check CHECK (amount >= 0)
);

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


-- Every gateway notification (simulated in 5c). event_id is UNIQUE, so the
-- same notification delivered twice is stored once and processed once.
CREATE TABLE payment_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID        NOT NULL REFERENCES payments (id) ON DELETE RESTRICT,
    event_id   UUID        NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    payload    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_events_event_id_key UNIQUE (event_id),
    CONSTRAINT payment_events_type_check CHECK (event_type IN ('payment.succeeded', 'payment.failed'))
);

CREATE INDEX payment_events_payment_idx ON payment_events (payment_id, created_at);


-- Stock that was sold and comes back after a refund (5c) gets its own type.
ALTER TABLE inventory_movements DROP CONSTRAINT inventory_movements_type_check;
ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_type_check
    CHECK (movement_type IN ('adjustment', 'reservation', 'release', 'consumption', 'restock'));
