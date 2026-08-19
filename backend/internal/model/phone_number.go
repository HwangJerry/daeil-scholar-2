package model

import "strings"

// CanonicalPhoneNumber keeps only ASCII decimal digits. Phone numbers cross
// several legacy boundaries in this application, so comparisons and new writes
// must use the same representation regardless of display punctuation.
type CanonicalPhoneNumber string

func NormalizePhoneNumber(value string) CanonicalPhoneNumber {
	var normalized strings.Builder
	normalized.Grow(len(value))
	for _, character := range value {
		if character >= '0' && character <= '9' {
			normalized.WriteRune(character)
		}
	}
	return CanonicalPhoneNumber(normalized.String())
}

func (p CanonicalPhoneNumber) String() string {
	return string(p)
}
