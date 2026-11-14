package addresses

import "time"

// AddressResponse mirrors Address in frontend/src/types/address.ts.
type AddressResponse struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	RecipientName string    `json:"recipientName"`
	Phone         string    `json:"phone"`
	AddressLine   string    `json:"addressLine"`
	City          string    `json:"city"`
	Province      string    `json:"province"`
	PostalCode    string    `json:"postalCode"`
	IsDefault     bool      `json:"isDefault"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func NewAddressResponse(a Address) AddressResponse {
	return AddressResponse{
		ID: a.ID.String(), Label: a.Label, RecipientName: a.RecipientName, Phone: a.Phone,
		AddressLine: a.AddressLine, City: a.City, Province: a.Province, PostalCode: a.PostalCode,
		IsDefault: a.IsDefault, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func NewAddressListResponse(list []Address) []AddressResponse {
	out := make([]AddressResponse, 0, len(list))
	for _, a := range list {
		out = append(out, NewAddressResponse(a))
	}
	return out
}

// CreateAddressRequest is POST /api/v1/me/addresses.
type CreateAddressRequest struct {
	Label         string `json:"label"`
	RecipientName string `json:"recipientName"`
	Phone         string `json:"phone"`
	AddressLine   string `json:"addressLine"`
	City          string `json:"city"`
	Province      string `json:"province"`
	PostalCode    string `json:"postalCode"`
	IsDefault     bool   `json:"isDefault"`
}

func (r CreateAddressRequest) fields() Fields {
	return Fields{
		Label: r.Label, RecipientName: r.RecipientName, Phone: r.Phone, AddressLine: r.AddressLine,
		City: r.City, Province: r.Province, PostalCode: r.PostalCode,
	}
}

// UpdateAddressRequest is PATCH /api/v1/me/addresses/:id. Omitted = unchanged.
// There is no isDefault here; use POST /me/addresses/:id/default.
type UpdateAddressRequest struct {
	Label         *string `json:"label"`
	RecipientName *string `json:"recipientName"`
	Phone         *string `json:"phone"`
	AddressLine   *string `json:"addressLine"`
	City          *string `json:"city"`
	Province      *string `json:"province"`
	PostalCode    *string `json:"postalCode"`
}
