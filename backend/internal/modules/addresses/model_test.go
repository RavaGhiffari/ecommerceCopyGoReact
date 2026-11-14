package addresses

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func validFields() Fields {
	return Fields{
		Label: "Rumah", RecipientName: "Budi Santoso", Phone: "0812-3456 7890",
		AddressLine: "Jl. Melati No. 5, RT 01/RW 02", City: "Bandung", Province: "Jawa Barat", PostalCode: "40115",
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"0812-3456 7890":   "081234567890",
		" (021) 555.1234 ": "0215551234",
		"+62 812 3456 789": "+628123456789",
	}
	for in, want := range cases {
		assert.Equal(t, want, NormalizePhone(in), in)
	}
}

func TestClean_TrimsAndNormalises(t *testing.T) {
	f := validFields()
	f.City = "  Bandung  "
	out, err := f.Clean()
	require.NoError(t, err)
	assert.Equal(t, "Bandung", out.City)
	assert.Equal(t, "081234567890", out.Phone)
}

func TestClean_ReportsEveryBadField(t *testing.T) {
	out, err := Fields{Phone: "12ab", PostalCode: "4011"}.Clean()
	_ = out
	var fields validation.Errors
	require.ErrorAs(t, err, &fields)
	for _, k := range []string{"label", "recipientName", "addressLine", "city", "province", "phone", "postalCode"} {
		assert.Contains(t, fields, k)
	}
}

func TestClean_PhoneAndPostalCodeFormats(t *testing.T) {
	bad := []struct{ phone, postal string }{
		{"1234567", "40115"},          // 7 digits: too short
		{"1234567890123456", "40115"}, // 16 digits: too long
		{"++6281234567", "40115"},     // two plus signs
		{"081234567890", "4011a"},     // postal code with a letter
		{"081234567890", "401150"},    // postal code too long
	}
	for _, c := range bad {
		f := validFields()
		f.Phone, f.PostalCode = c.phone, c.postal
		_, err := f.Clean()
		assert.Error(t, err, "%+v", c)
	}
}
