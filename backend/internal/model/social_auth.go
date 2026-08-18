package model

import "time"

type SocialProvider string

const (
	SocialProviderKakao SocialProvider = "KT"
	SocialProviderApple SocialProvider = "AP"
)

func (p SocialProvider) Valid() bool {
	return p == SocialProviderKakao || p == SocialProviderApple
}

type VerifiedSocialIdentity struct {
	Provider      SocialProvider
	Subject       string
	Email         string
	EmailVerified bool
}

type SocialProviderProfile struct {
	DisplayName     string `json:"displayName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	Email           string `json:"email,omitempty"`
	ProfileImageURL string `json:"profileImageUrl,omitempty"`
}

type SocialAuthorization interface {
	Provider() SocialProvider
}

type KakaoAuthorization struct {
	AccessToken string
}

func (KakaoAuthorization) Provider() SocialProvider {
	return SocialProviderKakao
}

type AppleAuthorization struct {
	ChallengeID       string
	IdentityToken     string
	AuthorizationCode string
	GivenName         string
	FamilyName        string
}

func (AppleAuthorization) Provider() SocialProvider {
	return SocialProviderApple
}

type MobileSession struct {
	User             AuthUser `json:"user"`
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	AccessIssuedAt   int64    `json:"accessIssuedAt"`
	AccessExpiresAt  int64    `json:"accessExpiresAt"`
	RefreshExpiresAt int64    `json:"refreshExpiresAt"`
	SID              string   `json:"sid"`
	JTI              string   `json:"jti"`
}

type SocialAuthStatus string

const (
	SocialAuthAuthenticated SocialAuthStatus = "authenticated"
	SocialAuthLinkRequired  SocialAuthStatus = "linkRequired"
	SocialAuthPending       SocialAuthStatus = "pending"
	SocialAuthRejected      SocialAuthStatus = "rejected"
)

type SocialAuthResult struct {
	Status       SocialAuthStatus     `json:"status"`
	Session      *MobileSession       `json:"session,omitempty"`
	LinkRequired *SocialLinkContext   `json:"linkRequired,omitempty"`
	Pending      *AuthUser            `json:"pending,omitempty"`
	Rejected     *SocialAuthRejection `json:"rejected,omitempty"`
}

type SocialLinkContext struct {
	LinkToken string                `json:"linkToken"`
	Provider  SocialProvider        `json:"provider"`
	Profile   SocialProviderProfile `json:"profile"`
	ExpiresAt int64                 `json:"expiresAt"`
}

type SocialAuthRejection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AppleChallenge struct {
	ID        string
	Nonce     string
	NonceHash string
	ExpiresAt time.Time
}
