// Package validate provides a tiny, dependency-free validation helper that
// accumulates field-level errors. It is deliberately minimal: request structs
// call these helpers in a Validate() method and return the collected map.
package validate

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Validator accumulates field errors keyed by JSON field name.
type Validator struct {
	Errors map[string]string
}

func New() *Validator { return &Validator{Errors: map[string]string{}} }

func (v *Validator) Valid() bool { return len(v.Errors) == 0 }

// add records the first error seen for a field.
func (v *Validator) add(field, msg string) {
	if _, exists := v.Errors[field]; !exists {
		v.Errors[field] = msg
	}
}

func (v *Validator) Check(ok bool, field, msg string) {
	if !ok {
		v.add(field, msg)
	}
}

func (v *Validator) Required(field, value string) {
	v.Check(strings.TrimSpace(value) != "", field, "is required")
}

func (v *Validator) Email(field, value string) {
	_, err := mail.ParseAddress(value)
	v.Check(err == nil, field, "must be a valid email address")
}

func (v *Validator) MinLen(field, value string, n int) {
	v.Check(utf8.RuneCountInString(value) >= n, field, "is too short")
}

func (v *Validator) MaxLen(field, value string, n int) {
	v.Check(utf8.RuneCountInString(value) <= n, field, "is too long")
}

// NormalizeEmail lower-cases and trims an email for consistent storage/lookup.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
