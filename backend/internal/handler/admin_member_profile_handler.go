// admin_member_profile_handler.go — HTTP endpoint for administrators correcting member profile fields
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

// UpdateProfile handles PUT /api/admin/member/{seq}/profile.
func (h *AdminMemberHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req model.AdminMemberProfileUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.UpdateProfile(seq, req); err != nil {
		writeMemberProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeMemberProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrMemberNameRequired):
		respondError(w, http.StatusBadRequest, "INVALID_NAME", "이름을 100자 이내로 입력해주세요")
	case errors.Is(err, service.ErrInvalidPhone):
		respondError(w, http.StatusBadRequest, "INVALID_PHONE", "휴대폰 번호 형식이 올바르지 않습니다")
	case errors.Is(err, service.ErrPhoneTaken):
		respondError(w, http.StatusConflict, "PHONE_TAKEN", "다른 회원이 사용 중인 휴대폰 번호입니다")
	case errors.Is(err, service.ErrInvalidMemberEmail):
		respondError(w, http.StatusBadRequest, "INVALID_EMAIL", "이메일 형식이 올바르지 않습니다")
	case errors.Is(err, service.ErrInvalidMemberCohort):
		respondError(w, http.StatusBadRequest, "INVALID_COHORT", "기수는 1~99 사이 숫자로 입력해주세요")
	case errors.Is(err, service.ErrInvalidDepartment):
		respondError(w, http.StatusBadRequest, "INVALID_DEPARTMENT", "학과를 목록에서 선택해주세요")
	case errors.Is(err, service.ErrMemberNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "회원을 찾을 수 없습니다")
	default:
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "회원 정보 수정에 실패했습니다")
	}
}
