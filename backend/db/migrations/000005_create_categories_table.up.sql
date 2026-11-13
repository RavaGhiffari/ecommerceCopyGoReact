-- Product categories, at most two levels deep:
--   Elektronik            (parent_id IS NULL)
--   └── Audio             (parent_id = Elektronik)
CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT: a category that still has children cannot be deleted.
    parent_id  UUID         REFERENCES categories (id) ON DELETE RESTRICT,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    sort_order INTEGER      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT categories_not_own_parent_check CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT categories_slug_format_check CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT categories_name_not_blank_check CHECK (LENGTH(TRIM(name)) > 0)
);

CREATE INDEX categories_parent_id_idx ON categories (parent_id);

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The two-level rule cannot be a CHECK constraint (a CHECK only sees its own
-- row, not the parent's), so a trigger enforces it. The service checks the same
-- rule first to give a friendly error; this is the last line of defence.
CREATE FUNCTION categories_enforce_two_levels()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.parent_id IS NOT NULL THEN
        -- The parent must itself be top-level...
        IF EXISTS (SELECT 1 FROM categories WHERE id = NEW.parent_id AND parent_id IS NOT NULL) THEN
            RAISE EXCEPTION 'category % would be nested more than two levels deep', NEW.slug
                USING ERRCODE = 'check_violation', CONSTRAINT = 'categories_max_two_levels';
        END IF;
        -- ...and a category that has children cannot become a child.
        IF EXISTS (SELECT 1 FROM categories WHERE parent_id = NEW.id) THEN
            RAISE EXCEPTION 'category % has children and cannot get a parent', NEW.slug
                USING ERRCODE = 'check_violation', CONSTRAINT = 'categories_max_two_levels';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER categories_two_levels
    BEFORE INSERT OR UPDATE OF parent_id ON categories
    FOR EACH ROW EXECUTE FUNCTION categories_enforce_two_levels();
