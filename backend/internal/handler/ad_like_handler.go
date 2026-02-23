// ad_like_handler.go — HTTP handler for ad like toggle endpoint
package handler

import (
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// AdLikeHandler handles ad like HTTP requests.
type AdLikeHandler struct {
	service *service.AdLikeService
}

// NewAdLikeHandler creates a new AdLikeHandler.
func NewAdLikeHandler(svc *service.AdLikeService) *AdLikeHandler {
	return &AdLikeHandler{service: svc}
}

// ToggleLike handles POST /api/ad/{maSeq}/like.
func (h *AdLikeHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	maSeq, err := strconv.Atoi(chi.URLParam(r, "maSeq"))
	if err != nil || maSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid maSeq")
		return
	}

	user, authErr := middleware.GetAuthUserOrError(r.Context())
	if authErr != nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "인증 정보가 없습니다")
		return
	}

	result, err := h.service.ToggleLike(maSeq, user.USRSeq)
	if err != nil {
		log.Error().Err(err).Int("maSeq", maSeq).Msg("ad like toggle failed")
		respondError(w, http.StatusInternalServerError, "LIKE_FAILED", "좋아요 처리에 실패했습니다")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
