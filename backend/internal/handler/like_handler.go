// like_handler.go — HTTP handler for like toggle endpoint
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// LikeHandler handles like-related HTTP requests.
type LikeHandler struct {
	service *service.LikeService
}

// NewLikeHandler creates a new LikeHandler.
func NewLikeHandler(svc *service.LikeService) *LikeHandler {
	return &LikeHandler{service: svc}
}

// ToggleLike handles POST /api/feed/{seq}/like.
func (h *LikeHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}

	user, err := middleware.GetAuthUserOrError(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	result, err := h.service.ToggleLike(seq, user.USRSeq, user.USRName)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("like toggle failed")
		respondError(w, http.StatusInternalServerError, "LIKE_FAILED", "좋아요 처리에 실패했습니다")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
