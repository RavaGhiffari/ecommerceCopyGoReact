package products

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		in   string
		want *string
	}{
		{"", nil},
		{"   !!! ", nil},
		{"Kemeja", ptr("kemeja:*")},
		{"  kemeja   BATIK ", ptr("kemeja:* & batik:*")},
		// tsquery operators are stripped, so input can never break the query
		{"kemeja & !batik | (x) 'y':*", ptr("kemeja:* & batik:* & x:* & y:*")},
		{"20.000 mAh", ptr("20:* & 000:* & mah:*")},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, buildSearchQuery(tt.in), "input %q", tt.in)
	}
}

func TestBuildSearchQuery_LimitsWords(t *testing.T) {
	got := buildSearchQuery(strings.Repeat("a ", 20))
	require.NotNil(t, got)
	assert.Equal(t, maxSearchWords, strings.Count(*got, ":*"))
}

func TestCheckPrice(t *testing.T) {
	tests := []struct {
		in      string
		wantErr string
	}{
		{"349000", ""},
		{"349000.50", ""},
		{"0", ""},
		{"-1", "cannot be negative"},
		{"1.234", "may have at most 2 decimal places"},
		{"10000000000", "is too large"},
	}
	for _, tt := range tests {
		errs := validation.Errors{}
		d := decimal.RequireFromString(tt.in)
		checkPrice(errs, "price", &d)
		if tt.wantErr == "" {
			assert.Empty(t, errs, tt.in)
		} else {
			assert.Equal(t, tt.wantErr, errs["price"], tt.in)
		}
	}

	errs := validation.Errors{}
	checkPrice(errs, "price", nil)
	assert.Equal(t, "is required", errs["price"])
}

func TestCleanSKU(t *testing.T) {
	errs := validation.Errors{}
	assert.Equal(t, "KMJ-01_A.2", cleanSKU(errs, "sku", "  kmj-01_a.2 "))
	assert.Empty(t, errs)

	for _, bad := range []string{"", "-start", "has space", "emoji🙂", strings.Repeat("A", 65)} {
		errs := validation.Errors{}
		cleanSKU(errs, "sku", bad)
		assert.Contains(t, errs, "sku", "input %q", bad)
	}
}

func TestCleanImages_OnlyHTTPURLs(t *testing.T) {
	errs := validation.Errors{}
	out := cleanImages(errs, []ImageInput{
		{URL: "https://cdn.example.com/a.jpg", Alt: "depan"},
		{URL: "javascript:alert(1)"},
		{URL: "data:image/png;base64,AAAA"},
		{URL: "/relative.jpg"},
	})
	assert.Len(t, out, 1)
	assert.Contains(t, errs, "images[1].url")
	assert.Contains(t, errs, "images[2].url")
	assert.Contains(t, errs, "images[3].url")
}

func ptr(s string) *string { return &s }
