package model

import "testing"

func TestNormalizePhoneNumberKeepsOnlyASCIIDigits(t *testing.T) {
	tests := map[string]string{
		"010-1234-5678":      "01012345678",
		" 010 1234 5678 ":    "01012345678",
		"+82 (10) 1234-5678": "821012345678",
		"010１２３４5678":        "0105678",
		"not-a-phone-number": "",
	}

	for input, expected := range tests {
		if actual := NormalizePhoneNumber(input).String(); actual != expected {
			t.Fatalf("NormalizePhoneNumber(%q) = %q, want %q", input, actual, expected)
		}
	}
}
