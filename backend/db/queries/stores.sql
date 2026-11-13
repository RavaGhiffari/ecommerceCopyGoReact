-- name: CreateStore :one
INSERT INTO stores (owner_id, name, slug, description, location)
VALUES (@owner_id, @name, @slug, @description, @location)
RETURNING id, owner_id, name, slug, description, location, status, created_at, updated_at;

-- name: GetStoreByOwner :one
SELECT id, owner_id, name, slug, description, location, status, created_at, updated_at
FROM stores
WHERE owner_id = @owner_id;

-- name: GetActiveStoreBySlug :one
-- Public store page. Only products that are active AND have a variant count
-- (the same rule the public product list uses).
SELECT s.id, s.owner_id, s.name, s.slug, s.description, s.location, s.status, s.created_at, s.updated_at,
       (SELECT COUNT(*)
        FROM products p
        WHERE p.store_id = s.id
          AND p.status = 'active'
          AND EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id))::int AS active_product_count
FROM stores s
WHERE s.slug = @slug
  AND s.status = 'active';

-- name: UpdateStore :one
UPDATE stores
SET name = @name, description = @description, location = @location
WHERE id = @id
RETURNING id, owner_id, name, slug, description, location, status, created_at, updated_at;
