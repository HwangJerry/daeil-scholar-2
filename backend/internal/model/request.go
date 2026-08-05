// request.go — API request DTOs for handler layer input binding.
package model

// LoginRequest is the request body for POST /api/auth/login (legacy ID/PW).
type LoginRequest struct {
	USRID    string `json:"usrId"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// KakaoMobileLoginRequest is the request body for POST /api/auth/kakao/mobile.
type KakaoMobileLoginRequest struct {
	GrantType   string `json:"grantType"` // access_token only
	AccessToken string `json:"accessToken"`
}

// RefreshTokenRequest is the request body for POST /api/auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
