-- Products belong to one store and one category. Prices live on variants:
-- every product has at least one variant (enforced by the service, which
-- creates product + variants in one transaction), and listings show the
-- lowest variant price.
CREATE TABLE products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT on both: products are archived, never deleted, so order history
    -- (later phases) can always point at them.
    store_id    UUID         NOT NULL REFERENCES stores (id) ON DELETE RESTRICT,
    category_id UUID         NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    name        VARCHAR(200) NOT NULL,
    slug        VARCHAR(200) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    -- Denormalised counter for "popular" sorting; incremented by checkout later.
    sold_count  INTEGER      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Full-text search document, kept up to date by PostgreSQL itself.
    -- 'simple' = no language-specific stemming (there is no built-in
    -- Indonesian dictionary); the name weighs more than the description.
    search      TSVECTOR GENERATED ALWAYS AS (
                    setweight(to_tsvector('simple', name), 'A') ||
                    setweight(to_tsvector('simple', description), 'B')
                ) STORED,

    CONSTRAINT products_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT products_slug_format_check CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT products_name_not_blank_check CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT products_sold_count_check CHECK (sold_count >= 0),
    -- Slugs are unique per store: two stores may both sell "kaos-polos".
    CONSTRAINT products_store_slug_key UNIQUE (store_id, slug)
);

-- Indexes follow the actual queries in db/queries/products.sql:
CREATE INDEX products_search_idx ON products USING GIN (search);          -- ?q=
CREATE INDEX products_category_status_idx ON products (category_id, status); -- category pages
CREATE INDEX products_status_created_idx ON products (status, created_at DESC); -- "newest"
CREATE INDEX products_status_sold_idx ON products (status, sold_count DESC);     -- "popular"
-- (store_id, slug) is already indexed by the UNIQUE constraint and also serves "products of a store".

CREATE TRIGGER products_set_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


CREATE TABLE product_variants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID          NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    sku        VARCHAR(64)   NOT NULL UNIQUE,
    name       VARCHAR(100)  NOT NULL,
    -- NUMERIC, never float: 0.1 + 0.2 must equal 0.3 when money is involved.
    price      NUMERIC(12,2) NOT NULL,
    sort_order INTEGER       NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT product_variants_price_check CHECK (price >= 0),
    CONSTRAINT product_variants_name_not_blank_check CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT product_variants_product_name_key UNIQUE (product_id, name)
);

CREATE INDEX product_variants_product_id_idx ON product_variants (product_id);

CREATE TRIGGER product_variants_set_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();


-- Images are links only for now (no file upload yet). Without images the UI
-- draws a placeholder.
CREATE TABLE product_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- CASCADE is fine here: images are not part of order history.
    product_id UUID         NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        TEXT         NOT NULL,
    alt        VARCHAR(200) NOT NULL DEFAULT '',
    sort_order INTEGER      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Only web URLs: blocks javascript: and data: URLs from ever reaching an <img src>.
    CONSTRAINT product_images_url_check CHECK (url ~ '^https?://')
);

CREATE INDEX product_images_product_id_idx ON product_images (product_id, sort_order);
