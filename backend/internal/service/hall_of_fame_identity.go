// hall_of_fame_identity.go — Phone HMAC and natural_key derivation; raw phones stop here.
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

const (
	hallOfFameKoreaCountryCode = "82"
	hallOfFameMinPhoneDigits   = 9
	hallOfFameMaxPhoneDigits   = 11
)

var hallOfFamePhoneShapePattern = regexp.MustCompile(`^[0-9+\-(). ]+$`)

// hallOfFameIdentity derives stable identity values. Rotating the secret changes
// every phone-based natural_key, so it must stay fixed once data is imported.
type hallOfFameIdentity struct{ secret []byte }

func newHallOfFameIdentity(secret string) hallOfFameIdentity {
	return hallOfFameIdentity{secret: []byte("hall-of-fame-phone:v1:" + secret)}
}

// canonicalHallOfFamePhone returns domestic digits starting with 0, or "" for
// overseas numbers and text such as "해외거주", which get no hash by SPEC.
func canonicalHallOfFamePhone(raw string) string {
	value := strings.TrimSpace(raw)
	if !hallOfFamePhoneShapePattern.MatchString(value) {
		return ""
	}
	digits := model.NormalizePhoneNumber(value).String()
	if strings.HasPrefix(digits, hallOfFameKoreaCountryCode+"1") {
		digits = "0" + strings.TrimPrefix(digits, hallOfFameKoreaCountryCode)
	} else if strings.HasPrefix(digits, "1") {
		// A numeric spreadsheet cell drops the leading zero of 010 numbers.
		digits = "0" + digits
	}
	if !strings.HasPrefix(digits, "0") || len(digits) < hallOfFameMinPhoneDigits || len(digits) > hallOfFameMaxPhoneDigits {
		return ""
	}
	return digits
}

func (i hallOfFameIdentity) phoneHash(rawPhone string) *string {
	phone := canonicalHallOfFamePhone(rawPhone)
	if phone == "" {
		return nil
	}
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write([]byte(phone))
	hash := hex.EncodeToString(mac.Sum(nil))
	return &hash
}

func normalizeHallOfFameName(name string) string {
	return strings.ToLower(strings.ReplaceAll(normalizeHallOfFameText(name), " ", ""))
}

func hallOfFameCohortKey(cohort *int) string {
	if cohort == nil {
		return "0"
	}
	return strconv.Itoa(*cohort)
}

// hallOfFameNaturalKey is sha256(name|cohort|phone_hash), or name|cohort|kind when
// no phone hash exists. Kind is excluded from phone keys so reclassifying a person
// updates the row instead of creating a second one.
func hallOfFameNaturalKey(name string, cohort *int, kind model.HallOfFameKind, phoneHash *string) string {
	discriminator := string(kind)
	if phoneHash != nil {
		discriminator = *phoneHash
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", normalizeHallOfFameName(name), hallOfFameCohortKey(cohort), discriminator)))
	return hex.EncodeToString(sum[:])
}
