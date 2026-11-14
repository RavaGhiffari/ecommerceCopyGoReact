package shipping

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSameCity(t *testing.T) {
	assert.True(t, SameCity("Bandung", "bandung"))
	assert.True(t, SameCity("  Jakarta   Selatan ", "jakarta selatan"))
	assert.False(t, SameCity("Jakarta Selatan", "Jakarta"), "no fuzzy matching: free text compared as is")
	assert.False(t, SameCity("", ""), "empty never matches")
}

func TestOptions_SameDayOnlyInTheSameCity(t *testing.T) {
	r := DefaultRates()

	same := r.Options("Bandung", "BANDUNG")
	assert.Equal(t, []Method{Regular, Express, SameDay}, methods(same))

	other := r.Options("Bandung", "Jakarta Selatan")
	assert.Equal(t, []Method{Regular, Express}, methods(other))

	assert.True(t, r.Allowed(SameDay, "Bandung", "bandung"))
	assert.False(t, r.Allowed(SameDay, "Bandung", "Depok"))
	assert.True(t, r.Allowed(Express, "Bandung", "Depok"))
	assert.False(t, r.Allowed(Method("drone"), "Bandung", "Bandung"))
}

func TestFees(t *testing.T) {
	r := DefaultRates()
	for m, want := range map[Method]string{Regular: "15000", Express: "30000", SameDay: "45000"} {
		fee, ok := r.Fee(m)
		assert.True(t, ok)
		assert.True(t, fee.Equal(decimal.RequireFromString(want)), "%s: %s", m, fee)
	}
	_, ok := r.Fee("teleport")
	assert.False(t, ok)
}

func TestValid(t *testing.T) {
	assert.True(t, Valid("same_day"))
	assert.False(t, Valid("SAME_DAY"))
	assert.False(t, Valid(""))
}

func methods(opts []Option) []Method {
	out := make([]Method, len(opts))
	for i, o := range opts {
		out[i] = o.Method
	}
	return out
}
