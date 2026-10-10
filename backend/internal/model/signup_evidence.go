package model

import "time"

// SignupEvidence is prepared by the server after validating signup requirements.
// It is never decoded from client JSON and is persisted in the member transaction.
type SignupEvidence struct {
	PhoneGrantHash string
	Consent        *SignupConsent
}

type SignupConsent struct {
	Version    string
	AcceptedAt time.Time
}
