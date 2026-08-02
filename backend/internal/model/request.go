// request.go — API request DTOs for handler layer input binding.
package model

// LoginRequest is the request body for POST /api/auth/login (legacy ID/PW).
type LoginRequest struct {
	Email    string `json:"email"`
	USRID    string `json:"usrId"`
	Password string `json:"password"`
}

type KakaoMobileLoginRequest struct {
	GrantType   string `json:"grantType"`
	AccessToken string `json:"accessToken"`
	Code        string `json:"code"`
	RedirectURI string `json:"redirectUri"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AppleMobileLoginRequest struct {
	ChallengeID       string `json:"challengeId"`
	IdentityToken     string `json:"identityToken"`
	AuthorizationCode string `json:"authorizationCode"`
	GivenName         string `json:"givenName"`
	FamilyName        string `json:"familyName"`
}
