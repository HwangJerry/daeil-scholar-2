package model

import "strings"

// CanonicalPhoneNumber keeps only ASCII decimal digits. Phone numbers cross
// several legacy boundaries in this application, so comparisons and new writes
// must use the same representation regardless of display punctuation.
type CanonicalPhoneNumber string

const (
	MinCanonicalPhoneDigits = 7
	MaxCanonicalPhoneDigits = 15
	LegacyPhoneDigitsLimit  = 11
)

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

func (p CanonicalPhoneNumber) Valid() bool {
	length := len(p)
	return length >= MinCanonicalPhoneDigits && length <= MaxCanonicalPhoneDigits
}
