package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func TestValidateRegister_NormalizesInput(t *testing.T) {
	in, err := validateRegister(RegisterInput{Name: "  Rava  ", Email: "  Rava@Example.COM ", Password: "12345678"})
	require.NoError(t, err)
	assert.Equal(t, "Rava", in.Name)
	assert.Equal(t, "rava@example.com", in.Email)
}

func TestValidateRegister_ReportsEveryField(t *testing.T) {
	_, err := validateRegister(RegisterInput{Name: " ", Email: "not-an-email", Password: "short"})

	var fields validation.Errors
	require.ErrorAs(t, err, &fields)
	assert.Equal(t, "is required", fields["name"])
	assert.Equal(t, "must be a valid email address", fields["email"])
	assert.Equal(t, "must be at least 8 characters", fields["password"])
}

func TestValidateRegister_PasswordLengthCountsCharactersNotBytes(t *testing.T) {
	// 8 characters, 16 bytes in UTF-8.
	_, err := validateRegister(RegisterInput{Name: "A", Email: "a@b.co", Password: "ééééééé1"})
	assert.NoError(t, err)

	_, err = validateRegister(RegisterInput{Name: "A", Email: "a@b.co", Password: strings.Repeat("x", 129)})
	var fields validation.Errors
	require.ErrorAs(t, err, &fields)
	assert.Equal(t, "must be at most 128 characters", fields["password"])
}

func TestValidateRegister_RejectsDisplayNameEmails(t *testing.T) {
	_, err := validateRegister(RegisterInput{Name: "A", Email: "Rava <rava@example.com>", Password: "12345678"})
	var fields validation.Errors
	require.ErrorAs(t, err, &fields)
	assert.Contains(t, fields, "email")
}
