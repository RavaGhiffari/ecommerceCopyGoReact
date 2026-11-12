package users

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func TestCleanPhone(t *testing.T) {
	tests := []struct {
		in      string
		want    *string
		wantErr bool
	}{
		{in: "", want: nil},
		{in: "   ", want: nil},
		{in: " +62 812-3456-7890 ", want: ptr("+62 812-3456-7890")},
		{in: "(021) 555 1234", want: ptr("(021) 555 1234")},
		{in: "0812abc4567", wantErr: true},
		{in: "12345", wantErr: true},
		{in: "1234567890123456789012345678901", wantErr: true},
	}
	for _, tt := range tests {
		errs := validation.Errors{}
		got := cleanPhone(errs, "phone", tt.in)
		if tt.wantErr {
			assert.Contains(t, errs, "phone", "input %q", tt.in)
			continue
		}
		assert.Empty(t, errs, "input %q", tt.in)
		assert.Equal(t, tt.want, got, "input %q", tt.in)
	}
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "rava@example.com", NormalizeEmail("  Rava@Example.COM\t"))
}

func ptr(s string) *string { return &s }
