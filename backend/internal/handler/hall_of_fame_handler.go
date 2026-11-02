// hall_of_fame_handler.go — Public Hall of Fame wall with HTTP caching.
package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog/log"
)

const hallOfFameCacheControl = "public, max-age=300"

type HallOfFameHandler struct {
	service *service.HallOfFameService
}

func NewHallOfFameHandler(hallOfFameService *service.HallOfFameService) *HallOfFameHandler {
	return &HallOfFameHandler{service: hallOfFameService}
}

func (h *HallOfFameHandler) Get(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.GetPublic()
	if err != nil {
		log.Error().Err(err).Msg("hall of fame load failed")
		respondError(w, http.StatusInternalServerError, "HALL_OF_FAME_FAILED", "명예의 전당을 불러오지 못했습니다.")
		return
	}
	body, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("hall of fame encode failed")
		respondError(w, http.StatusInternalServerError, "HALL_OF_FAME_FAILED", "명예의 전당을 불러오지 못했습니다.")
		return
	}
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	w.Header().Set("Cache-Control", hallOfFameCacheControl)
	w.Header().Set("ETag", etag)
	if strings.Contains(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
