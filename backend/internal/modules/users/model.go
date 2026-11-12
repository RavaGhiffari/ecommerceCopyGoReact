// Package users owns user accounts and profiles.
//
// Layers (same pattern every business module follows):
//
//	handler.go    HTTP only: read the request, call the service, write the response
//	service.go    business rules (validation, which roles, what may change)
//	repository.go data access via sqlc-generated queries; no rules
//	dto.go        JSON request/response shapes (the API contract)
//	model.go      the domain type passed between the layers
package users

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database/queries"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

// User never contains the password hash: the only code that sees the hash is
// the login path in the auth module.
type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Phone     *string
	Status    string
	Roles     []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// fromRow maps the sqlc row type. CreateUserRow and UpdateUserProfileRow have
// identical fields, so callers convert them with GetUserByIDRow(row).
func fromRow(r queries.GetUserByIDRow) User {
	return User{
		ID:        r.ID,
		Email:     r.Email,
		Name:      r.Name,
		Phone:     r.Phone,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// ---- input rules shared with the auth module --------------------------------

// NormalizeEmail trims and lowercases, matching the database CHECK constraint.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail expects an already normalized email.
func ValidateEmail(errs validation.Errors, field, email string) {
	switch {
	case email == "":
		errs.Add(field, "is required")
	case len(email) > 255:
		errs.Add(field, "must be at most 255 characters")
	default:
		// ParseAddress also accepts `Name <a@b.c>`; require the bare address.
		addr, err := mail.ParseAddress(email)
		if err != nil || addr.Address != email {
			errs.Add(field, "must be a valid email address")
		}
	}
}

// CleanName trims and validates a display name, returning the trimmed value.
func CleanName(errs validation.Errors, field, name string) string {
	name = strings.TrimSpace(name)
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		errs.Add(field, "is required")
	case n > 150:
		errs.Add(field, "must be at most 150 characters")
	}
	return name
}

// cleanPhone accepts digits, spaces, '+', '-' and parentheses. Empty means
// "no phone number" and is stored as NULL.
func cleanPhone(errs validation.Errors, field, phone string) *string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil
	}
	if len(phone) > 30 {
		errs.Add(field, "must be at most 30 characters")
		return nil
	}
	digits := 0
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == ' ' || r == '+' || r == '-' || r == '(' || r == ')':
		default:
			errs.Add(field, "may only contain digits, spaces and + - ( )")
			return nil
		}
	}
	if digits < 8 {
		errs.Add(field, "must contain at least 8 digits")
		return nil
	}
	return &phone
}
