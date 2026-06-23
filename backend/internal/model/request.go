// request.go — API request DTOs for handler layer input binding.
package model

// LoginRequest is the request body for POST /api/auth/login (legacy ID/PW).
type LoginRequest struct {
	USRID    string `json:"usrId"`
	Password string `json:"password"`
}

// KakaoMobileLoginRequest is the request body for POST /api/auth/kakao/mobile.
type KakaoMobileLoginRequest struct {
	GrantType   string `json:"grantType"` // access_token | authorization_code
	AccessToken string `json:"accessToken"`
	Code        string `json:"code"`
	RedirectURI string `json:"redirectUri"`
}

// RefreshTokenRequest is the request body for POST /api/auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
