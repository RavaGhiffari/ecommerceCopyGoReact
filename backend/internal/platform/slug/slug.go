// Package slug turns names into URL-friendly identifiers:
// "Kemeja Batik Pria (Parang)!" -> "kemeja-batik-pria-parang".
//
// The same pattern is enforced by CHECK constraints in the database
// (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'), so an invalid slug cannot be stored
// even if this code is bypassed.
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const MaxLength = 80

var valid = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Make derives a slug from free text. Accents are dropped ("é" -> "e");
// anything that is not a-z or 0-9 becomes a single hyphen. The result may be
// empty (e.g. for "!!!"), which callers must treat as invalid.
func Make(s string) string {
	var b strings.Builder
	hyphen := false
	// NFD splits "é" into "e" + a combining accent, which we then skip.
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			hyphen = false
		case unicode.Is(unicode.Mn, r): // combining mark (the accent)
		default:
			if !hyphen && b.Len() > 0 {
				b.WriteByte('-')
				hyphen = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > MaxLength {
		out = strings.TrimRight(out[:MaxLength], "-")
	}
	return out
}

// Valid reports whether s is already a well-formed slug.
func Valid(s string) bool {
	return len(s) <= MaxLength && valid.MatchString(s)
}
