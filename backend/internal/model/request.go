// request.go — API request DTOs for handler layer input binding.
package model

// LoginRequest is the request body for POST /api/auth/login (legacy ID/PW).
type LoginRequest struct {
	USRID    string `json:"usrId"`
	Password string `json:"password"`
}

// RefreshTokenRequest is the request body for POST /api/auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
