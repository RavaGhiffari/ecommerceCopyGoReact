-- One store per seller (UNIQUE owner_id). The owner gets the 'seller' role in
-- the same transaction that creates the store.
CREATE TABLE stores (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT: deleting a user must not silently delete a store with orders.
    owner_id    UUID         NOT NULL UNIQUE REFERENCES users (id) ON DELETE RESTRICT,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         NOT NULL DEFAULT '',
    location    VARCHAR(100) NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT stores_status_check CHECK (status IN ('active', 'suspended')),
    CONSTRAINT stores_slug_format_check CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT stores_name_not_blank_check CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT stores_location_not_blank_check CHECK (LENGTH(TRIM(location)) > 0)
);

CREATE TRIGGER stores_set_updated_at
    BEFORE UPDATE ON stores
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
