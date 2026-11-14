// Package addresses manages a buyer's saved shipping addresses (Phase 5a).
//
// Rules:
//   - An address always belongs to the session's user; every query filters by
//     user_id, so someone else's address id is simply "not found".
//   - At most MaxPerUser addresses per user.
//   - When a user has addresses, exactly one is the default. The database
//     guarantees "at most one" (partial unique index); this package keeps
//     "at least one": the first address becomes the default, and deleting the
//     default promotes the newest remaining address.
//   - Checkout (5b) copies an address onto each order, so changing or deleting
//     an address never rewrites history.
package addresses

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

const MaxPerUser = 10

var (
	ErrNotFound     = errors.New("address not found")
	ErrLimitReached = errors.New("address limit reached")
)

type Address struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Label         string
	RecipientName string
	Phone         string
	AddressLine   string
	City          string
	Province      string
	PostalCode    string
	IsDefault     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Fields is the editable part of an address, used by create and update.
type Fields struct {
	Label         string
	RecipientName string
	Phone         string
	AddressLine   string
	City          string
	Province      string
	PostalCode    string
}

// Same patterns as the CHECK constraints in migration 000010. The service
// checks first so the user gets a field error (422) instead of a 500.
var (
	phonePattern  = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
	postalPattern = regexp.MustCompile(`^[0-9]{5}$`)
	// Separators people type in phone numbers: "0812-3456 7890", "(021) 555.1234".
	phoneSeparators = strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "")
)

// NormalizePhone removes common separators so "0812-3456 7890" is stored as
// "081234567890". It does not change the country format (08… stays 08…).
func NormalizePhone(s string) string {
	return phoneSeparators.Replace(strings.TrimSpace(s))
}

// Clean trims and normalises every field and reports all problems at once,
// using the JSON field names as keys.
func (f Fields) Clean() (Fields, error) {
	errs := validation.Errors{}
	out := Fields{
		Label:         cleanText(errs, "label", f.Label, 1, 50),
		RecipientName: cleanText(errs, "recipientName", f.RecipientName, 2, 100),
		AddressLine:   cleanText(errs, "addressLine", f.AddressLine, 5, 255),
		City:          cleanText(errs, "city", f.City, 2, 100),
		Province:      cleanText(errs, "province", f.Province, 2, 100),
		Phone:         NormalizePhone(f.Phone),
		PostalCode:    strings.TrimSpace(f.PostalCode),
	}
	switch {
	case out.Phone == "":
		errs.Add("phone", "is required")
	case !phonePattern.MatchString(out.Phone):
		errs.Add("phone", "must be 8-15 digits, optionally starting with +")
	}
	switch {
	case out.PostalCode == "":
		errs.Add("postalCode", "is required")
	case !postalPattern.MatchString(out.PostalCode):
		errs.Add("postalCode", "must be 5 digits")
	}
	return out, errs.Err()
}

func cleanText(errs validation.Errors, field, v string, minLen, maxLen int) string {
	v = strings.TrimSpace(v)
	switch n := utf8.RuneCountInString(v); {
	case n == 0 && minLen > 0:
		errs.Add(field, "is required")
	case n < minLen:
		errs.Add(field, "is too short")
	case n > maxLen:
		errs.Add(field, "is too long")
	}
	return v
}
