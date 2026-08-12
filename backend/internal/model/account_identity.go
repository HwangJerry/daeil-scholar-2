package model

import (
	"errors"
	"strings"
)

var (
	ErrIdentitySubjectRequired     = errors.New("identity subject is required")
	ErrNormalizedEmailProvider     = errors.New("email metadata is only for EMAIL identity")
	ErrEmailNotVerified            = errors.New("email identity must be verified")
	ErrUnsupportedIdentityProvider = errors.New("unsupported identity provider")
)

type IdentityProvider string
type IdentityStatus string

const (
	IdentityProviderEmail         IdentityProvider = "EMAIL"
	IdentityProviderKakao         IdentityProvider = "KAKAO"
	IdentityProviderApple         IdentityProvider = "APPLE"
	IdentityProviderLocalUsername IdentityProvider = "LOCAL_USERNAME"
	IdentityStatusActive          IdentityStatus   = "ACTIVE"
	IdentityStatusDisabled        IdentityStatus   = "DISABLED"
	IdentityStatusRevoked         IdentityStatus   = "REVOKED"
)

func (p IdentityProvider) SupportsPassword() bool {
	switch p {
	case IdentityProviderEmail, IdentityProviderLocalUsername:
		return true
	default:
		return false
	}
}

func (p IdentityProvider) SupportsProviderCredential() bool {
	switch p {
	case IdentityProviderKakao, IdentityProviderApple:
		return true
	default:
		return false
	}
}

type Identity struct {
	IdentityID      int64            `json:"identityId" db:"IDENTITY_ID"`
	AccountSeq      int              `json:"accountSeq" db:"ACCOUNT_ID"`
	Provider        IdentityProvider `json:"provider" db:"PROVIDER"`
	SubjectKey      string           `json:"subjectKey" db:"SUBJECT_KEY"`
	NormalizedEmail *string          `json:"normalizedEmail,omitempty" db:"NORMALIZED_EMAIL"`
	Status          IdentityStatus   `json:"status" db:"STATUS"`
}

func NewIdentity(accountSeq int, provider IdentityProvider, subjectKey string, metadata string, emailVerified bool) (Identity, error) {
	subjectKey = strings.TrimSpace(subjectKey)
	if accountSeq <= 0 {
		return Identity{}, ErrIdentitySubjectRequired
	}
	if subjectKey == "" {
		return Identity{}, ErrIdentitySubjectRequired
	}

	normalizedMetadata := strings.ToLower(strings.TrimSpace(metadata))

	switch provider {
	case IdentityProviderEmail:
		if !emailVerified {
			return Identity{}, ErrEmailNotVerified
		}
		if normalizedMetadata == "" {
			return Identity{}, ErrNormalizedEmailProvider
		}
		return Identity{
			AccountSeq:      accountSeq,
			Provider:        provider,
			SubjectKey:      normalizedMetadata,
			NormalizedEmail: &normalizedMetadata,
			Status:          IdentityStatusActive,
		}, nil
	case IdentityProviderKakao, IdentityProviderApple, IdentityProviderLocalUsername:
		if normalizedMetadata != "" {
			return Identity{}, ErrNormalizedEmailProvider
		}
		return Identity{
			AccountSeq: accountSeq,
			Provider:   provider,
			SubjectKey: subjectKey,
			Status:     IdentityStatusActive,
		}, nil
	default:
		return Identity{}, ErrUnsupportedIdentityProvider
	}
}
