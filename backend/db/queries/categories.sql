-- name: ListCategories :many
-- Parents first, then children; the client builds the tree from parent_id.
SELECT id, parent_id, name, slug, sort_order
FROM categories
ORDER BY parent_id NULLS FIRST, sort_order, name;

-- name: GetCategoryByID :one
SELECT id, parent_id, name, slug, sort_order
FROM categories
WHERE id = @id;

-- name: GetCategoryBySlug :one
SELECT id, parent_id, name, slug, sort_order
FROM categories
WHERE slug = @slug;

-- name: CategoryHasChildren :one
SELECT EXISTS (SELECT 1 FROM categories WHERE parent_id = @id::uuid);

-- name: ListCategoryTreeIDs :many
-- A category plus its direct children: filtering by "Elektronik" must also
-- return products filed under "Audio".
SELECT id FROM categories WHERE id = @id::uuid OR parent_id = @id::uuid;

-- name: CreateCategory :one
INSERT INTO categories (parent_id, name, slug, sort_order)
VALUES (@parent_id, @name, @slug, @sort_order)
RETURNING id, parent_id, name, slug, sort_order;

-- name: UpdateCategory :one
UPDATE categories
SET parent_id = @parent_id, name = @name, slug = @slug, sort_order = @sort_order
WHERE id = @id
RETURNING id, parent_id, name, slug, sort_order;
