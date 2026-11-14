package orders

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Same pattern as the orders_order_number_format_check constraint.
var numberPattern = regexp.MustCompile(`^PK-[0-9]{8}-[0-9A-HJKMNP-TV-Z]{6}$`)

func TestNewOrderNumber_Format(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for range 1000 {
		n := NewOrderNumber(now)
		assert.Regexp(t, numberPattern, n)
		assert.Equal(t, "PK-20261114-", n[:12])
		seen[n] = true
	}
	assert.Greater(t, len(seen), 990, "random suffixes should almost never repeat")
}

func TestNewOrderNumber_UsesIndonesianDate(t *testing.T) {
	// 18:30 UTC on the 14th is already 01:30 WIB on the 15th.
	late := time.Date(2026, 11, 14, 18, 30, 0, 0, time.UTC)
	assert.Equal(t, "PK-20261115-", NewOrderNumber(late)[:12])
}
