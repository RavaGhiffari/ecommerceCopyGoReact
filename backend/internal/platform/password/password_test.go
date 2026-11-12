package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cheap parameters keep the tests fast; production uses DefaultParams.
var testParams = Params{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1}

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("correct horse battery staple", testParams)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$m=8192,t=1,p=1$"))
	assert.NotContains(t, hash, "correct horse", "the hash must not contain the password")

	ok, err := Verify("correct horse battery staple", hash)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = Verify("Correct horse battery staple", hash)
	require.NoError(t, err)
	assert.False(t, ok, "verification is case-sensitive")
}

func TestHash_SamePasswordGivesDifferentHashes(t *testing.T) {
	a, err := Hash("same-password", testParams)
	require.NoError(t, err)
	b, err := Hash("same-password", testParams)
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "a random salt per hash means equal passwords are not visible in the table")
}

func TestVerify_UsesParametersStoredInTheHash(t *testing.T) {
	stronger := Params{MemoryKiB: 16 * 1024, Iterations: 2, Parallelism: 1}
	hash, err := Hash("pw-12345678", stronger)
	require.NoError(t, err)

	// Verify takes no Params argument: it must read them from the hash, so old
	// hashes keep working after the configured cost changes.
	ok, err := Verify("pw-12345678", hash)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerify_RejectsMalformedHashes(t *testing.T) {
	for _, bad := range []string{
		"",
		"plaintext",
		"$argon2i$v=19$m=8192,t=1,p=1$c2FsdA$aGFzaA",  // wrong algorithm
		"$argon2id$v=18$m=8192,t=1,p=1$c2FsdA$aGFzaA", // wrong version
		"$argon2id$v=19$m=8192,t=0,p=1$c2FsdA$aGFzaA", // zero iterations would panic in argon2
		"$argon2id$v=19$m=8192,t=1,p=0$c2FsdA$aGFzaA", // zero threads would panic in argon2
		"$argon2id$v=19$m=8192,t=1,p=1$!!!$aGFzaA",    // bad base64
	} {
		ok, err := Verify("anything", bad)
		assert.ErrorIs(t, err, ErrMalformedHash, "input %q", bad)
		assert.False(t, ok)
	}
}
