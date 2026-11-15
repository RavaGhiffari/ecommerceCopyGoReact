DROP TABLE IF EXISTS reviews;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_rating_check,
    DROP COLUMN IF EXISTS rating_total,
    DROP COLUMN IF EXISTS rating_count;
