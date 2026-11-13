package slug

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMake(t *testing.T) {
	tests := map[string]string{
		"Kemeja Batik Pria (Parang)!":         "kemeja-batik-pria-parang",
		"  Power Bank 20.000 mAh  ":           "power-bank-20-000-mah",
		"Café Crème":                          "cafe-creme",
		"Gadget & Wearable":                   "gadget-wearable",
		"---a---b---":                         "a-b",
		"!!!":                                 "",
		"Rumah--Asri":                         "rumah-asri",
	}
	for in, want := range tests {
		got := Make(in)
		assert.Equal(t, want, got, "Make(%q)", in)
		if got != "" {
			assert.True(t, Valid(got), "Make(%q) must produce a valid slug", in)
		}
	}
}

func TestMake_TruncatesWithoutTrailingHyphen(t *testing.T) {
	got := Make(strings.Repeat("ab ", 60))
	assert.LessOrEqual(t, len(got), MaxLength)
	assert.True(t, Valid(got))
}

func TestValid(t *testing.T) {
	for _, ok := range []string{"a", "kemeja-batik", "20-000"} {
		assert.True(t, Valid(ok), ok)
	}
	for _, bad := range []string{"", "-a", "a-", "a--b", "Kemeja", "a b", "a_b", strings.Repeat("a", MaxLength+1)} {
		assert.False(t, Valid(bad), bad)
	}
}
