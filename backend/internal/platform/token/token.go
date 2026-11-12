// Package token creates session tokens.
//
// The raw token goes to the browser in a cookie; only its SHA-256 hash is
// stored. A fast hash is fine here (unlike passwords): the token is 256 random
// bits, so there is nothing to guess or brute-force, and hashing it only makes
// sure a leaked database cannot be replayed as cookies.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const byteLength = 32 // 256 bits of randomness

// New returns a random URL-safe token and the hash to store for it.
func New() (raw string, hash []byte, err error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("token: read random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, Hash(raw), nil
}

// Hash is what the database stores and looks up for a raw token.
func Hash(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
