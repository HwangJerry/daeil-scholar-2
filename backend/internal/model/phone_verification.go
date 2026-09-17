// phone_verification.go — DTOs and stored record for SMS phone-ownership verification at signup.
package model

import "time"

// PhoneVerificationRequest starts a verification: an SMS code is sent to this number.
type PhoneVerificationRequest struct {
	Phone string `json:"phone"`
}

// PhoneVerificationRequestResult is returned after a code has been dispatched.
// VerificationID is the public handle the client echoes back when confirming.
type PhoneVerificationRequestResult struct {
	VerificationID string `json:"verificationId"`
	ExpiresInSec   int    `json:"expiresInSec"`
}

// PhoneVerificationConfirmRequest submits the code the user received by SMS.
type PhoneVerificationConfirmRequest struct {
	VerificationID string `json:"verificationId"`
	Code           string `json:"code"`
}

// PhoneVerificationConfirmResult carries the short-lived grant token proving the
// phone number was verified. Registration requires it and consumes it once.
type PhoneVerificationConfirmResult struct {
	VerificationToken string `json:"verificationToken"`
	ExpiresInSec      int    `json:"expiresInSec"`
}

// PhoneVerification is a stored verification attempt. Neither the SMS code nor the
// grant token is kept in plaintext; only their SHA-256 hashes are persisted.
type PhoneVerification struct {
	APVSeq         int64      `db:"APV_SEQ"`
	APVID          string     `db:"APV_ID"`
	Phone          string     `db:"PHONE"`
	CodeHash       string     `db:"CODE_HASH"`
	GrantTokenHash *string    `db:"GRANT_TOKEN_HASH"`
	Attempts       int        `db:"ATTEMPTS"`
	VerifiedYN     string     `db:"VERIFIED_YN"`
	ConsumedYN     string     `db:"CONSUMED_YN"`
	ExpiresAt      time.Time  `db:"EXPIRES_AT"`
	GrantExpiresAt *time.Time `db:"GRANT_EXPIRES_AT"`
	RegDate        time.Time  `db:"REG_DATE"`
}

// SMSMessage is a single outbound text message handed to an SMS provider.
type SMSMessage struct {
	To   string
	Body string
}
