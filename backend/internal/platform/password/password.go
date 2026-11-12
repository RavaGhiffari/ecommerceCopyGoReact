// Package password hashes and verifies passwords with Argon2id.
//
// Why Argon2id: it is deliberately slow AND memory-hungry, so an attacker who
// steals the users table cannot try billions of guesses per second on GPUs.
// A plain SHA-256 would be far too fast for passwords.
//
// The output is a self-describing "PHC string":
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
//
// The parameters travel with every hash, so Verify always uses the settings a
// hash was created with. Raising the cost later does not break old accounts.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params controls how expensive one hash is.
type Params struct {
	MemoryKiB   uint32 // RAM used per hash
	Iterations  uint32 // passes over that memory
	Parallelism uint8  // threads
}

// DefaultParams follows the OWASP Password Storage Cheat Sheet minimum for
// Argon2id (19 MiB, 2 iterations, 1 thread): roughly tens of milliseconds per
// hash on a server, acceptable for login, painful for an attacker.
var DefaultParams = Params{MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1}

const (
	saltLength = 16 // bytes; unique per hash so equal passwords get different hashes
	keyLength  = 32 // bytes of derived key
)

var ErrMalformedHash = errors.New("password: malformed hash")

var b64 = base64.RawStdEncoding

// Hash returns the PHC string for plain. Each call uses a fresh random salt.
func Hash(plain string, p Params) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, p.Iterations, p.MemoryKiB, p.Parallelism, keyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKiB, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify reports whether plain matches the encoded hash.
// A wrong password is (false, nil); an error means the stored hash is broken.
func Verify(plain, encoded string) (bool, error) {
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}

	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallelism); err != nil {
		return false, ErrMalformedHash
	}
	// argon2.IDKey panics on nonsense parameters; refuse them instead.
	if p.Iterations == 0 || p.Parallelism == 0 || p.MemoryKiB < 8*uint32(p.Parallelism) {
		return false, ErrMalformedHash
	}

	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, ErrMalformedHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrMalformedHash
	}

	got := argon2.IDKey([]byte(plain), salt, p.Iterations, p.MemoryKiB, p.Parallelism, uint32(len(want)))

	// Constant-time comparison: how long the check takes must not reveal how
	// many leading bytes matched.
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
