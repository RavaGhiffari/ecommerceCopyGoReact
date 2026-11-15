package reviews

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

func TestInputClean(t *testing.T) {
	got, err := Input{Rating: 5, Comment: "  Bagus sekali  "}.Clean()
	require.NoError(t, err)
	assert.Equal(t, "Bagus sekali", got.Comment, "comment is trimmed")

	_, err = Input{Rating: 4}.Clean()
	assert.NoError(t, err, "a comment is optional")

	for _, r := range []int{0, 6, -1} {
		_, err := Input{Rating: r}.Clean()
		var fields validation.Errors
		require.True(t, errors.As(err, &fields), "rating %d", r)
		assert.Contains(t, fields, "rating")
	}

	// 1000 characters (runes, not bytes) is the limit.
	ok := strings.Repeat("é", MaxCommentLength)
	_, err = Input{Rating: 3, Comment: ok}.Clean()
	assert.NoError(t, err, "1000 multi-byte characters are allowed")
	_, err = Input{Rating: 3, Comment: ok + "x"}.Clean()
	var fields validation.Errors
	require.True(t, errors.As(err, &fields))
	assert.Contains(t, fields, "comment")
}

func TestMaskName(t *testing.T) {
	cases := map[string]string{
		"Rava Ghiffari":       "Rava G***",
		"  Siti   Nur Aisyah": "Siti N***",
		"Budi":                "B***",
		"élise martin":        "élise m***",
		"Ömer":                "Ö***",
		"":                    "Pembeli",
		"   ":                 "Pembeli",
	}
	for in, want := range cases {
		assert.Equal(t, want, MaskName(in), "%q", in)
	}
}

func TestNotAllowedErrorUnwraps(t *testing.T) {
	err := error(&NotAllowedError{OrderStatus: "shipped"})
	assert.ErrorIs(t, err, ErrNotAllowed)
}
