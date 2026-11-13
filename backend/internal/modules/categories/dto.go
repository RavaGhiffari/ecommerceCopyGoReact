package categories

import "github.com/google/uuid"

// CategoryResponse is the public JSON shape. Mirrored by Category in
// frontend/src/types/marketplace.ts.
type CategoryResponse struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parentId"` // null for top-level
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	SortOrder int32   `json:"sortOrder"`
}

func NewCategoryResponse(c Category) CategoryResponse {
	var parent *string
	if c.ParentID != nil {
		s := c.ParentID.String()
		parent = &s
	}
	return CategoryResponse{ID: c.ID.String(), ParentID: parent, Name: c.Name, Slug: c.Slug, SortOrder: c.SortOrder}
}

// CategoryRequest is the body of POST /admin/categories and
// PUT /admin/categories/:id (a full definition; omitted parentId = top-level).
type CategoryRequest struct {
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	ParentID  *uuid.UUID `json:"parentId"`
	SortOrder int32      `json:"sortOrder"`
}
