package products

import (
	"strings"
	"unicode"
)

const (
	maxSearchWords   = 8
	maxSearchWordLen = 50
)

// buildSearchQuery turns what a user typed into a PostgreSQL tsquery string:
//
//	"Kemeja  batik!"  ->  "kemeja:* & batik:*"
//
// Every word must match (&), and ":*" makes each word a prefix, so "kem"
// already finds "kemeja" while the user is still typing.
//
// Why this is safe: tsquery has its own syntax (& | ! ( ) : * < > '), and
// malformed input makes to_tsquery() raise an error. We keep only letters and
// digits, so the result is always valid and the user cannot inject operators.
// The string is still passed as a bind parameter ($1), never concatenated
// into SQL.
//
// Returns nil when nothing searchable remains (no filter).
func buildSearchQuery(q string) *string {
	isSeparator := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }

	var terms []string
	for _, w := range strings.FieldsFunc(strings.ToLower(q), isSeparator) {
		if len(terms) == maxSearchWords {
			break
		}
		if r := []rune(w); len(r) > maxSearchWordLen {
			w = string(r[:maxSearchWordLen])
		}
		terms = append(terms, w+":*")
	}
	if len(terms) == 0 {
		return nil
	}
	s := strings.Join(terms, " & ")
	return &s
}
