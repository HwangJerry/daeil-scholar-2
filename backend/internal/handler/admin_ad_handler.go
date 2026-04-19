// Admin ad handler — HTTP lifecycle for ad management endpoints
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminAdHandler struct {
	service *service.AdminAdService
}

func NewAdminAdHandler(svc *service.AdminAdService) *AdminAdHandler {
	return &AdminAdHandler{service: svc}
}

// parseUTCISOtoDB converts "2026-03-17T00:00:00.000Z" → "2026-03-17 00:00:00" for DB DATETIME.
func parseUTCISOtoDB(iso string) (string, error) {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return "", err
	}
	return t.UTC().Format("2006-01-02 15:04:05"), nil
}

// adRequest is the JSON shape received from the admin frontend.
type adRequest struct {
	MAName       string  `json:"maName"`
	MAURL        string  `json:"maUrl"`
	MAImg        string  `json:"maImg"`
	MAStatus     string  `json:"maStatus"`
	ADTier       string  `json:"adTier"`
	ADTitleLabel string  `json:"adTitleLabel"`
	MAIndx       int     `json:"maIndx"`
	ADStartDate  *string `json:"adStartDate"`
	ADEndDate    *string `json:"adEndDate"`
}

func (h *AdminAdHandler) toInsert(req *adRequest) (*model.AdminAdInsert, error) {
	ins := &model.AdminAdInsert{
		MAName:       req.MAName,
		MAURL:        req.MAURL,
		MAImg:        req.MAImg,
		MAStatus:     req.MAStatus,
		ADTier:       req.ADTier,
		ADTitleLabel: req.ADTitleLabel,
		MAIndx:       req.MAIndx,
	}
	if req.ADStartDate != nil && *req.ADStartDate != "" {
		dbVal, err := parseUTCISOtoDB(*req.ADStartDate)
		if err != nil {
			return nil, err
		}
		ins.ADStartDate = &dbVal
	}
	if req.ADEndDate != nil && *req.ADEndDate != "" {
		dbVal, err := parseUTCISOtoDB(*req.ADEndDate)
		if err != nil {
			return nil, err
		}
		ins.ADEndDate = &dbVal
	}
	return ins, nil
}

func (h *AdminAdHandler) List(w http.ResponseWriter, r *http.Request) {
	ads, err := h.service.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list ads")
		return
	}
	respondJSON(w, http.StatusOK, ads)
}

func (h *AdminAdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req adRequest
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
		if errors.Is(err, service.ErrTierConflict) {
			respondError(w, http.StatusConflict, "TIER_CONFLICT", "이미 활성화된 동일 등급의 광고가 있습니다.")
			return
		}
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create ad")
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int{"maSeq": seq})
}

func (h *AdminAdHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req adRequest
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
		if errors.Is(err, service.ErrTierConflict) {
			respondError(w, http.StatusConflict, "TIER_CONFLICT", "이미 활성화된 동일 등급의 광고가 있습니다.")
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAdHandler) Delete(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	if err := h.service.Delete(seq); err != nil {
		respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete ad")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAdHandler) Stats(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(r.URL.Query().Get("maSeq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Missing maSeq")
		return
	}
	stats, err := h.service.GetStats(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to get ad stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
