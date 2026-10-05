// feed_inline_detail_handler.go — include=detail parsing and the explicit post view counter (POST /api/feed/{seq}/view)
package handler

import (
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// inlineDetailInclude is the `include` value that asks feed endpoints to carry
// each post's expanded detail (body, files, comments).
const inlineDetailInclude = "detail"

// wantsInlineDetail reports whether the request's comma-separated `include`
// query parameter names detail.
func wantsInlineDetail(r *http.Request) bool {
	for _, value := range r.URL.Query()["include"] {
		for _, part := range strings.Split(value, ",") {
			if strings.TrimSpace(part) == inlineDetailInclude {
				return true
			}
		}
	}
	return false
}

// RecordView handles POST /api/feed/{seq}/view: apps call it the first time a
// post is expanded in a session, since include=detail never counts a view.
func (h *FeedHandler) RecordView(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	hit, found, err := h.inlineService.RecordView(seq)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("feed view failed")
		respondError(w, http.StatusInternalServerError, "VIEW_FAILED", "Failed to record view")
		return
	}
	if !found {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
		return
	}
	respondJSON(w, http.StatusOK, model.NoticeViewResponse{Hit: hit})
}
