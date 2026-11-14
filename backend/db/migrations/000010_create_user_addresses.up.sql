-- A buyer's saved shipping addresses (Phase 5, decision Q1).
--
-- An address is NOT history: checkout copies the chosen address onto every
-- order, so editing or deleting an address later never changes an order.
-- That is why deleting a user may cascade here.
CREATE TABLE user_addresses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    label          VARCHAR(50)  NOT NULL,  -- e.g. "Rumah", "Kantor"
    recipient_name VARCHAR(100) NOT NULL,
    phone          VARCHAR(16)  NOT NULL,  -- normalised by the service: optional "+", then digits only
    address_line   VARCHAR(255) NOT NULL,
    city           VARCHAR(100) NOT NULL,
    province       VARCHAR(100) NOT NULL,
    postal_code    VARCHAR(5)   NOT NULL,
    is_default     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT user_addresses_phone_format_check CHECK (phone ~ '^\+?[0-9]{8,15}$'),
    CONSTRAINT user_addresses_postal_code_format_check CHECK (postal_code ~ '^[0-9]{5}$'),
    CONSTRAINT user_addresses_text_not_blank_check CHECK (
        LENGTH(TRIM(label)) > 0
        AND LENGTH(TRIM(recipient_name)) > 0
        AND LENGTH(TRIM(address_line)) > 0
        AND LENGTH(TRIM(city)) > 0
        AND LENGTH(TRIM(province)) > 0
    )
);

-- At most ONE default address per user, guaranteed by the database: a partial
-- unique index only contains the rows where is_default is true, so a second
-- default row for the same user is a unique violation. ("Exactly one default
-- when the user has addresses" is kept by the service.)
CREATE UNIQUE INDEX user_addresses_one_default_per_user_idx
    ON user_addresses (user_id)
    WHERE is_default;

-- "My addresses" lookups. The partial index above only covers default rows.
CREATE INDEX user_addresses_user_id_idx ON user_addresses (user_id, created_at DESC);

CREATE TRIGGER user_addresses_set_updated_at
    BEFORE UPDATE ON user_addresses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
