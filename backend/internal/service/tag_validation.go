// tag_validation.go — Validation rules for user-submitted tags.
package service

import (
	"errors"
	"strings"
)

// ErrTagContainsWhitespace is returned when a tag includes whitespace characters.
// Tags must be whitespace-free so that the alumni search can split keywords by spaces.
var ErrTagContainsWhitespace = errors.New("tag must not contain whitespace")

// ValidateTags returns ErrTagContainsWhitespace if any non-empty tag contains
// space, tab, carriage return, or newline. Empty strings are ignored (repo skips them on save).
func ValidateTags(tags []string) error {
	for _, t := range tags {
		if t == "" {
			continue
		}
		if strings.ContainsAny(t, " \t\r\n") {
			return ErrTagContainsWhitespace
		}
	}
	return nil
}
