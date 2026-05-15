// Disclosure handler — public read-only HTTP lifecycle for disclosure list/detail
package handler

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type DisclosureHandler struct {
	service   *service.DisclosureService
	presenter *presenter.FeedPresenter
}

func NewDisclosureHandler(svc *service.DisclosureService, pres *presenter.FeedPresenter) *DisclosureHandler {
	return &DisclosureHandler{service: svc, presenter: pres}
}

func (h *DisclosureHandler) GetList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := h.service.List(parseCursor(q.Get("cursor")), parseIntParam(q.Get("size")))
	if err != nil {
		log.Error().Err(err).Msg("disclosure list failed")
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to load disclosures")
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

func (h *DisclosureHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetDetail(seq)
	if err != nil {
		log.Error().Err(err).Int("seq", seq).Msg("disclosure detail failed")
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load detail")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
		return
	}
	respondJSON(w, http.StatusOK, h.presenter.FormatNoticeDetail(detail))
}
