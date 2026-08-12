package model

import (
	"encoding/base64"
)

type PasswordCredentialStatus string
type PasswordAlgorithm string

type ProviderCredential struct {
	CredentialID          int64                    `db:"CREDENTIAL_ID" json:"credentialId"`
	IdentityID            *int64                   `db:"IDENTITY_ID" json:"identityId,omitempty"`
	ContinuationTokenHash *string                  `db:"CONTINUATION_TOKEN_HASH" json:"continuationTokenHash,omitempty"`
	Provider              IdentityProvider         `db:"PROVIDER" json:"provider"`
	KeyID                 string                   `db:"KEY_ID" json:"keyId"`
	NonceBytes            []byte                   `db:"NONCE_BYTES" json:"-"`
	Algorithm             string                   `db:"ALGORITHM" json:"algorithm"`
	Ciphertext            []byte                   `db:"CIPHERTEXT" json:"ciphertext"`
	Status                PasswordCredentialStatus `db:"-" json:"-"`
}

const (
	PasswordCredentialStatusActive       PasswordCredentialStatus = "ACTIVE"
	PasswordCredentialStatusDisabled     PasswordCredentialStatus = "DISABLED"
	PasswordAlgorithmArgon2id            PasswordAlgorithm        = "ARGON2ID"
	PasswordAlgorithmMysqlNativePassword PasswordAlgorithm        = "MYSQL_NATIVE_PASSWORD"
)

// NonceString renders NonceBytes as base64url without padding for logs/telemetry.
func (c ProviderCredential) NonceString() string {
	if len(c.NonceBytes) == 0 {
		return ""
	}
	return base64.RawStdEncoding.EncodeToString(c.NonceBytes)
}

// PasswordCredential stores a canonical password-derived hash for AUTH_PASSWORD_CREDENTIAL rows.
type PasswordCredential struct {
	IdentityID     int64                    `db:"IDENTITY_ID" json:"identityId"`
	Provider       IdentityProvider         `db:"PROVIDER" json:"provider"`
	Algorithm      PasswordAlgorithm        `db:"ALGORITHM" json:"algorithm"`
	ParametersText *string                  `db:"PARAMETERS_TEXT" json:"parametersText,omitempty"`
	PasswordHash   string                   `db:"PASSWORD_HASH" json:"passwordHash"`
	Status         PasswordCredentialStatus `db:"STATUS" json:"status"`
}
