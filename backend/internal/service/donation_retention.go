// donation_retention.go — Validate record-specific Korean donation retention clocks.
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"time"
)

func ValidateDonationRetention(d model.DonationRetentionDecision) error {
	if d.OrderID <= 0 || strings.TrimSpace(d.Evidence) == "" || len(d.Evidence) > 200 {
		return errors.New("retention decision requires a record and review evidence")
	}
	years := 0
	switch d.Basis {
	case "ledger_10y":
		years = 10
	case "receipt_5y":
		years = 5
	case "external_original", "erase":
		if d.BasisDate != nil || d.Until != nil {
			return errors.New("non-archive decision must not invent a retention period")
		}
		return nil
	default:
		return errors.New("unknown donation retention basis")
	}
	if d.BasisDate == nil || d.Until == nil {
		return errors.New("retention clock is missing")
	}
	// Use calendar dates, not elapsed hours; the DB driver uses Asia/Seoul.
	date := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	if !date(*d.Until).Equal(retentionAnniversary(date(*d.BasisDate), years)) {
		return errors.New("retention end does not match the statutory clock")
	}
	return nil
}

// The archive intentionally uses a separate key from authentication credentials.
func DonationArchiveSealer(keyHex string) (func([]byte) ([]byte, error), error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, errors.New("donation archive requires a dedicated 32-byte hex key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return func(plain []byte) ([]byte, error) {
		nonce := make([]byte, aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return nil, err
		}
		return aead.Seal(nonce, nonce, plain, []byte("DFLH_DONATION_LEGAL_ARCHIVE_V1")), nil
	}, nil
}

// Calendar periods ending in a month without the starting day use its last day.
func retentionAnniversary(start time.Time, years int) time.Time {
	year := start.Year() + years
	lastDay := time.Date(year, start.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := start.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, start.Month(), day, 0, 0, 0, 0, time.UTC)
}

// DonationArchiveOpener reads the existing V1 nonce-prefixed archive format.
func DonationArchiveOpener(keyHex string) (func([]byte) ([]byte, error), error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, errors.New("donation archive requires a dedicated 32-byte hex key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return func(data []byte) ([]byte, error) {
		if len(data) < aead.NonceSize()+aead.Overhead() {
			return nil, errors.New("invalid archive ciphertext")
		}
		return aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte("DFLH_DONATION_LEGAL_ARCHIVE_V1"))
	}, nil
}
