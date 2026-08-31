// Identity transforms — converts legacy member data into canonical identity values.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	accountStatusActive    = "ACTIVE"
	accountStatusSuspended = "SUSPENDED"
	accountStatusWithdrawn = "WITHDRAWN"

	identityProviderLocalUsername = "LOCAL_USERNAME"
	identityProviderKakao         = "KAKAO"
	identityProviderApple         = "APPLE"

	identityStatusActive   = "ACTIVE"
	identityStatusDisabled = "DISABLED"

	legacyPasswordAlgorithm = "MYSQL_NATIVE_PASSWORD"
	maxIdentityKeyBytes     = 255
)

type identityMemberRow struct {
	AccountID    int    `db:"USR_SEQ"`
	Username     string `db:"USR_ID"`
	Email        string `db:"USR_EMAIL"`
	PasswordHash string `db:"USR_PWD"`
	LegacyStatus string `db:"USR_STATUS"`
}

type identitySocialRow struct {
	AccountID      int    `db:"USR_SEQ"`
	LegacyProvider string `db:"NMS_GATE"`
	SubjectKey     string `db:"NMS_ID"`
	LegacyStatus   string `db:"NMS_STATUS"`
}

type canonicalAccountState struct {
	Status      string
	SuspendedAt bool
	WithdrawnAt bool
}

type identityBackfillStats struct {
	MembersScanned             int
	SocialLinksScanned         int
	AccountStatesCreated       int
	PasswordIdentitiesCreated  int
	PasswordCredentialsCreated int
	SocialIdentitiesCreated    int
	ConflictCount              int
}

func (s identityBackfillStats) String() string {
	return fmt.Sprintf(
		"members=%d social_links=%d account_states=%d password_identities=%d password_credentials=%d social_identities=%d conflicts=%d",
		s.MembersScanned,
		s.SocialLinksScanned,
		s.AccountStatesCreated,
		s.PasswordIdentitiesCreated,
		s.PasswordCredentialsCreated,
		s.SocialIdentitiesCreated,
		s.ConflictCount,
	)
}

func mapLegacyAccountStatus(status string) canonicalAccountState {
	// LoginEligibilityPolicy is the current legacy authority: BAA (rejected
	// alumni verification), BBB (pending), CCC (approved), and ZZZ (root) may
	// all authenticate. AAA is the account-withdrawal marker. DDD and every
	// unknown or empty value are fail-closed as suspended.
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "BAA", "BBB", "CCC", "ZZZ":
		return canonicalAccountState{Status: accountStatusActive}
	case "AAA":
		return canonicalAccountState{Status: accountStatusWithdrawn, WithdrawnAt: true}
	default:
		return canonicalAccountState{Status: accountStatusSuspended, SuspendedAt: true}
	}
}

func normalizedEmail(raw string) *string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return nil
	}
	return &normalized
}

func sourceFingerprint(memberCount, maxAccountID int) string {
	sourceSummary := fmt.Sprintf("member_count=%d;max_usr_seq=%d", memberCount, maxAccountID)
	digest := sha256.Sum256([]byte(sourceSummary))
	return hex.EncodeToString(digest[:])
}

func sourceSummary(members []identityMemberRow) (int, int) {
	maxAccountID := 0
	for _, member := range members {
		if member.AccountID > maxAccountID {
			maxAccountID = member.AccountID
		}
	}
	return len(members), maxAccountID
}

func mapLegacySocialProvider(gate string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(gate)) {
	case "KT":
		return identityProviderKakao, true
	case "AP":
		return identityProviderApple, true
	default:
		return "", false
	}
}

func mapLegacySocialStatus(status string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "Y", "ACTIVE":
		return identityStatusActive, true
	case "N", "INACTIVE":
		return identityStatusDisabled, true
	default:
		return "", false
	}
}

func hasLegacyPassword(passwordHash string) bool {
	return strings.TrimSpace(passwordHash) != ""
}

func isMysqlNativePasswordHash(passwordHash string) bool {
	if len(passwordHash) != 41 || passwordHash[0] != '*' {
		return false
	}
	_, err := hex.DecodeString(passwordHash[1:])
	return err == nil
}

func isValidIdentityKey(value string) bool {
	if strings.TrimSpace(value) == "" || len(value) > maxIdentityKeyBytes {
		return false
	}
	for _, candidateByte := range []byte(value) {
		if candidateByte > 0x7f {
			return false
		}
	}
	return true
}

func isValidNormalizedEmail(email *string) bool {
	if email == nil {
		return true
	}
	return isValidIdentityKey(*email)
}

func incrementConflictCount(current int, isConflict bool) int {
	if !isConflict {
		return current
	}
	return current + 1
}
