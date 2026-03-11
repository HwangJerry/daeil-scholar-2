// profile_handler.go — HTTP handlers for user profile retrieval and update
package handler

import (
	"encoding/json"
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
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update profile")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
