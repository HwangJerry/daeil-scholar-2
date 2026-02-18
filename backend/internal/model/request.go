// request.go — API request DTOs for handler layer input binding.
package model

// LoginRequest is the request body for POST /api/auth/login (legacy ID/PW).
type LoginRequest struct {
	USRID    string `json:"usrId"`
	Password string `json:"password"`
}

