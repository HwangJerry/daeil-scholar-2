// comment_handler.go — HTTP handlers for comment CRUD endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// CommentHandler handles comment-related HTTP requests.
type CommentHandler struct {
	service *service.CommentService
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{service: svc}
}

// ListComments handles GET /api/feed/{seq}/comments.
func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}

	comments, err := h.service.GetComments(seq)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("list comments failed")
		respondError(w, http.StatusInternalServerError, "COMMENTS_FAILED", "댓글을 불러올 수 없습니다")
		return
	}

	respondJSON(w, http.StatusOK, comments)
}

// CreateComment handles POST /api/feed/{seq}/comments.
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}

	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
		return
	}

	user, err := middleware.GetAuthUserOrError(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	comment, err := h.service.AddComment(seq, user.USRSeq, user.USRName, req.Contents)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("create comment failed")
		respondError(w, http.StatusBadRequest, "COMMENT_FAILED", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, comment)
}

// DeleteComment handles DELETE /api/feed/{seq}/comments/{cSeq}.
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	cSeq := parseIntParam(chi.URLParam(r, "cSeq"))
	if cSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid comment seq")
		return
	}

	user, err := middleware.GetAuthUserOrError(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	if err := h.service.DeleteComment(cSeq, user.USRSeq); err != nil {
		log.Error().Err(err).Int("cSeq", cSeq).Msg("delete comment failed")
		respondError(w, http.StatusForbidden, "DELETE_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
