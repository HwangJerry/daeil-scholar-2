// profile_handler.go — HTTP handlers for user profile retrieval and update
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type ProfileHandler struct {
	service *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: svc}
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	profile, err := h.service.GetProfile(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "PROFILE_FAILED", "Failed to load profile")
		return
	}
	respondJSON(w, http.StatusOK, profile)
}

func (h *ProfileHandler) GetAlumniVerification(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	verification, err := h.service.GetAlumniVerification(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "VERIFICATION_LOAD_FAILED", "동문 인증 상태를 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, verification)
}

func (h *ProfileHandler) PutAlumniVerification(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var req model.AlumniVerificationSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.SubmitAlumniVerification(user.USRSeq, req); err != nil {
		switch {
		case errors.Is(err, service.ErrAcademicInformationRequired):
			respondError(w, http.StatusBadRequest, "ACADEMIC_INFORMATION_REQUIRED", "졸업연도, 기수, 학과를 모두 입력해주세요")
		case errors.Is(err, service.ErrInvalidDepartment):
			respondError(w, http.StatusBadRequest, "INVALID_DEPARTMENT", "유효하지 않은 학과입니다")
		default:
			respondError(w, http.StatusInternalServerError, "VERIFICATION_UPDATE_FAILED", "동문 인증 정보를 저장하지 못했습니다")
		}
		return
	}
	verification, err := h.service.GetAlumniVerification(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "VERIFICATION_LOAD_FAILED", "동문 인증 상태를 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, verification)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var req model.ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.UpdateProfile(user.USRSeq, req); err != nil {
		if errors.Is(err, service.ErrInvalidPhone) {
			respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
			return
		}
		if errors.Is(err, service.ErrPhoneTaken) {
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 등록된 전화번호입니다")
			return
		}
		if errors.Is(err, service.ErrTagContainsWhitespace) {
			respondError(w, http.StatusBadRequest, "INVALID_TAG", "태그에 공백을 포함할 수 없습니다")
			return
		}
		if errors.Is(err, service.ErrInvalidDepartment) {
			respondError(w, http.StatusBadRequest, "INVALID_DEPARTMENT", "유효하지 않은 학과입니다")
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update profile")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
