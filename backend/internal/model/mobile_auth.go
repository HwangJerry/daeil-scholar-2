// mobile_auth.go — Canonical mobile authentication response envelopes.
package model

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
)

type SocialAuthResult struct {
	Status       SocialAuthStatus   `json:"status"`
	Session      *MobileSession     `json:"session,omitempty"`
	LinkRequired *SocialLinkContext `json:"linkRequired,omitempty"`
}

type SocialProvider string

const SocialProviderKakao SocialProvider = "KT"

type SocialProviderProfile struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatarUrl"`
}

type SocialLinkContext struct {
	LinkToken string                `json:"linkToken"`
	Provider  SocialProvider        `json:"provider"`
	Profile   SocialProviderProfile `json:"profile"`
	ExpiresAt int64                 `json:"expiresAt"`
}
