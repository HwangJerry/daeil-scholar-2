package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminBannerAdHandler struct {
	service *service.AdminBannerAdService
}

func NewAdminBannerAdHandler(svc *service.AdminBannerAdService) *AdminBannerAdHandler {
	return &AdminBannerAdHandler{service: svc}
}

type bannerRequest struct {
	BNName      string   `json:"bnName"`
	BNURL       string   `json:"bnUrl"`
	OpenYN      string   `json:"openYn"`
	Indx        int      `json:"indx"`
	BNStartDate *string  `json:"bnStartDate"`
	BNEndDate   *string  `json:"bnEndDate"`
	ImageURLs   []string `json:"imageUrls"`
}

func (h *AdminBannerAdHandler) toInsert(req *bannerRequest) (*model.AdminBannerAdInsert, error) {
	ins := &model.AdminBannerAdInsert{
		BNName:    req.BNName,
		BNURL:     req.BNURL,
		OpenYN:    req.OpenYN,
		Indx:      req.Indx,
		ImageURLs: req.ImageURLs,
	}
	if req.BNStartDate != nil && *req.BNStartDate != "" {
		dbVal, err := parseUTCISOtoDB(*req.BNStartDate)
		if err != nil {
			return nil, err
		}
		ins.BNStartDate = &dbVal
	}
	if req.BNEndDate != nil && *req.BNEndDate != "" {
		dbVal, err := parseUTCISOtoDB(*req.BNEndDate)
		if err != nil {
			return nil, err
		}
		ins.BNEndDate = &dbVal
	}
	return ins, nil
}

func (h *AdminBannerAdHandler) List(w http.ResponseWriter, r *http.Request) {
	banners, err := h.service.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list banner ads")
		return
	}
	respondJSON(w, http.StatusOK, banners)
}

func (h *AdminBannerAdHandler) Detail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "bnSeq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid bnSeq")
		return
	}
	banner, err := h.service.Get(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to get banner ad")
		return
	}
	if banner == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Banner ad not found")
		return
	}
	respondJSON(w, http.StatusOK, banner)
}

func (h *AdminBannerAdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req bannerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	ins, err := h.toInsert(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid date format")
		return
	}
	seq, err := h.service.Create(ins)
	if err != nil {
		if h.respondMutationError(w, err) {
			return
		}
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create banner ad")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int{"bnSeq": seq})
}

func (h *AdminBannerAdHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req bannerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	ins, err := h.toInsert(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid date format")
		return
	}
	if err := h.service.Update(seq, ins); err != nil {
		if h.respondMutationError(w, err) {
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update banner ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminBannerAdHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete banner ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminBannerAdHandler) Stats(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(r.URL.Query().Get("bnSeq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Missing bnSeq")
		return
	}
	stats, err := h.service.GetStats(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to get banner ad stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

func (h *AdminBannerAdHandler) respondMutationError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, service.ErrInvalidBannerURL):
		respondError(w, http.StatusBadRequest, "INVALID_URL", "bnUrl must use http:// or https://")
	case errors.Is(err, service.ErrInvalidOpenYN):
		respondError(w, http.StatusBadRequest, "INVALID_OPEN_YN", "openYn must be Y or N")
	case errors.Is(err, service.ErrInvalidBannerPeriod):
		respondError(w, http.StatusBadRequest, "INVALID_PERIOD", "bnStartDate must not be later than bnEndDate")
	case errors.Is(err, service.ErrActiveWithoutImages):
		respondError(w, http.StatusBadRequest, "IMAGE_REQUIRED", "An active banner ad requires at least one image")
	case errors.Is(err, service.ErrActiveConflict):
		respondError(w, http.StatusConflict, "ACTIVE_CONFLICT", "게시기간이 겹치는 활성 배너 광고가 있습니다.")
	default:
		return false
	}
	return true
}
