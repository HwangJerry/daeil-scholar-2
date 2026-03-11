// password_reset.go — Password reset request/response models
package model

import "time"

// PasswordResetRequest is the JSON body for POST /api/auth/password/reset-request.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirm is the JSON body for POST /api/auth/password/reset-confirm.
type PasswordResetConfirm struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// PasswordResetToken represents a row in the ALUMNI_PASSWORD_RESET table.
type PasswordResetToken struct {
	APRSeq    int       `db:"APR_SEQ"`
	USRSeq    int       `db:"USR_SEQ"`
	Token     string    `db:"APR_TOKEN"`
	UsedYN    string    `db:"APR_USED_YN"`
	ExpiresAt time.Time `db:"EXPIRES_AT"`
	RegDate   time.Time `db:"REG_DATE"`
}

// ValidateTokenResponse is the JSON response for GET /api/auth/password/validate-token.
type ValidateTokenResponse struct {
	Valid bool   `json:"valid"`
	Name  string `json:"name,omitempty"`
}
