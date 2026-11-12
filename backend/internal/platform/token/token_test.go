package token

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	raw, hash, err := New()
	require.NoError(t, err)

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	require.NoError(t, err, "token must be URL-safe base64 so it fits in a cookie as-is")
	assert.Len(t, decoded, 32)
	assert.Len(t, hash, 32, "SHA-256; the DB CHECK constraint expects exactly 32 bytes")
	assert.Equal(t, Hash(raw), hash)
}

func TestNew_IsUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		raw, _, err := New()
		require.NoError(t, err)
		require.False(t, seen[raw])
		seen[raw] = true
	}
}

func TestHash_DependsOnEveryCharacter(t *testing.T) {
	assert.NotEqual(t, Hash("abc"), Hash("abd"))
	assert.Equal(t, Hash("abc"), Hash("abc"))
}
