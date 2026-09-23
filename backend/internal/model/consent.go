// consent.go — Signup consent DTO shared by ID/password registration and social linking.
package model

// ConsentTypePrivacy is the AUTH_CONSENT.CONSENT_TYPE value for the 개인정보 수집·이용 동의
// shown at signup.
const ConsentTypePrivacy = "PRIVACY"

// PrivacyConsent is what a client sends to record that the applicant accepted the
// data-collection notice. Version is the notice version the client displayed
// (the privacy policy publication date, e.g. "2026-09-30"). A nil pointer on the
// request means the client predates the consent UI.
type PrivacyConsent struct {
	Version  string `json:"version"`
	Accepted bool   `json:"accepted"`
}
