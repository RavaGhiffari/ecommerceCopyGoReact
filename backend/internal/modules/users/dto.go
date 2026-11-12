package users

import "time"

// UserResponse is the public JSON shape of a user (the API contract).
// Mirrored in frontend/src/types/auth.ts — keep the two in sync.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     *string   `json:"phone"` // null when not set
	Status    string    `json:"status"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewUserResponse(u User) UserResponse {
	roles := u.Roles
	if roles == nil {
		roles = []string{} // JSON [] rather than null
	}
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Phone:     u.Phone,
		Status:    u.Status,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
	}
}

// UpdateProfileRequest is the body of PATCH /api/v1/me.
// Omitted fields are not changed; "phone": "" removes the phone number.
type UpdateProfileRequest struct {
	Name  *string `json:"name"`
	Phone *string `json:"phone"`
}
