// ad_comment_handler.go — HTTP handlers for ad comment CRUD endpoints
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// AdCommentHandler handles ad comment HTTP requests.
type AdCommentHandler struct {
	service *service.AdCommentService
}

// NewAdCommentHandler creates a new AdCommentHandler.
func NewAdCommentHandler(svc *service.AdCommentService) *AdCommentHandler {
	return &AdCommentHandler{service: svc}
}

// ListComments handles GET /api/ad/{maSeq}/comments.
func (h *AdCommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	maSeq, err := strconv.Atoi(chi.URLParam(r, "maSeq"))
	if err != nil || maSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid maSeq")
		return
	}

	comments, err := h.service.ListComments(maSeq)
	if err != nil {
		log.Error().Err(err).Int("maSeq", maSeq).Msg("ad list comments failed")
		respondError(w, http.StatusInternalServerError, "COMMENTS_FAILED", "댓글을 불러올 수 없습니다")
		return
	}

	respondJSON(w, http.StatusOK, comments)
}

// CreateComment handles POST /api/ad/{maSeq}/comments.
func (h *AdCommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	maSeq, err := strconv.Atoi(chi.URLParam(r, "maSeq"))
	if err != nil || maSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid maSeq")
		return
	}

	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
		return
	}

	user, authErr := middleware.GetAuthUserOrError(r.Context())
	if authErr != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	comment, err := h.service.CreateComment(maSeq, user.USRSeq, user.USRName, req.Contents)
	if err != nil {
		log.Error().Err(err).Int("maSeq", maSeq).Msg("ad create comment failed")
		respondError(w, http.StatusBadRequest, "COMMENT_FAILED", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, comment)
}

// DeleteComment handles DELETE /api/ad/{maSeq}/comments/{acSeq}.
func (h *AdCommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	acSeq, err := strconv.Atoi(chi.URLParam(r, "acSeq"))
	if err != nil || acSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid comment seq")
		return
	}

	user, authErr := middleware.GetAuthUserOrError(r.Context())
	if authErr != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	if err := h.service.DeleteComment(acSeq, user.USRSeq); err != nil {
		log.Error().Err(err).Int("acSeq", acSeq).Msg("ad delete comment failed")
		respondError(w, http.StatusForbidden, "DELETE_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
