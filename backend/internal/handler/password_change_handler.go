// password_change_handler.go — HTTP handler for authenticated password change.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
)

type PasswordChangeHandler struct {
	svc *service.PasswordChangeService
}

func NewPasswordChangeHandler(svc *service.PasswordChangeService) *PasswordChangeHandler {
	return &PasswordChangeHandler{svc: svc}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *PasswordChangeHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "MISSING_FIELDS", "currentPassword and newPassword are required")
		return
	}

	if err := h.svc.ChangePassword(user.USRSeq, req.CurrentPassword, req.NewPassword); err != nil {
		switch err.Error() {
		case "NO_PASSWORD":
			respondError(w, http.StatusForbidden, "NO_PASSWORD", "카카오 로그인 계정은 비밀번호를 변경할 수 없습니다")
		case "WRONG_PASSWORD":
			respondError(w, http.StatusUnauthorized, "WRONG_PASSWORD", "현재 비밀번호가 올바르지 않습니다")
		default:
			respondError(w, http.StatusInternalServerError, "CHANGE_FAILED", "비밀번호 변경에 실패했습니다")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
