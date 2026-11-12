// Package validation collects per-field input errors.
//
// Services return a validation.Errors value (an error) and handlers turn it
// into HTTP 422 with the fields in "details". That keeps the rules in the
// service, where they apply no matter who calls it (HTTP handler, CLI seed
// command, tests), and keeps HTTP knowledge out of the service.
package validation

// Errors maps a field name to a human-readable problem.
type Errors map[string]string

func (e Errors) Error() string { return "validation failed" }

// Add records the first problem for a field; later ones are ignored so the
// user sees one clear message per field.
func (e Errors) Add(field, message string) {
	if _, exists := e[field]; !exists {
		e[field] = message
	}
}

// Err returns nil when there were no problems. Always return Err() (not the
// map itself) so an empty map does not become a non-nil error.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}
