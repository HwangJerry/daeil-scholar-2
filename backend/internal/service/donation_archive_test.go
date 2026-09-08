package service

import (
	"strings"
	"testing"
)

func TestDonationArchiveOpenerCompatibility(t *testing.T) {
	key := strings.Repeat("ab", 32)
	seal, _ := DonationArchiveSealer(key)
	open, _ := DonationArchiveOpener(key)
	encrypted, err := seal([]byte(`{"donorName":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := open(encrypted)
	if err != nil || string(plain) != `{"donorName":"test"}` {
		t.Fatal("round trip", err)
	}
	wrong, _ := DonationArchiveOpener(strings.Repeat("cd", 32))
	if _, err = wrong(encrypted); err == nil {
		t.Fatal("wrong key accepted")
	}
	encrypted[len(encrypted)-1] ^= 1
	if _, err = open(encrypted); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err = open([]byte{1}); err == nil {
		t.Fatal("short ciphertext accepted")
	}
	if _, err = DonationArchiveOpener(""); err == nil {
		t.Fatal("empty key accepted")
	}
}
