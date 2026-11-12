package auth

// RegisterRequest is the body of POST /api/v1/auth/register.
// There is no role field: every sign-up is a customer. Unknown fields (like
// "role") are rejected by request.BindJSON.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the body of POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
