// subscription_handler.go — HTTP handlers for subscription endpoints
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dflh-saf/backend/internal/config"
	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// SubscriptionHandler handles recurring donation subscription HTTP requests.
type SubscriptionHandler struct {
	subService    *service.SubscriptionService
	easypayConfig config.EasyPayConfig
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(subService *service.SubscriptionService, cfg config.EasyPayConfig) *SubscriptionHandler {
	return &SubscriptionHandler{subService: subService, easypayConfig: cfg}
}

// CreateSubscription handles POST /api/donation/subscription.
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
		return
	}

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}

	result, err := h.subService.CreateSubscription(user, req, ip, h.easypayConfig)
	if err != nil {
		respondError(w, http.StatusBadRequest, "SUBSCRIPTION_FAILED", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetMySubscription handles GET /api/donation/subscription.
func (h *SubscriptionHandler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	sub, err := h.subService.GetMySubscription(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "QUERY_FAILED", "조회 실패")
		return
	}
	respondJSON(w, http.StatusOK, sub)
}

// CancelSubscription handles DELETE /api/donation/subscription.
func (h *SubscriptionHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	var body struct {
		SubSeq int `json:"subSeq"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SubSeq == 0 {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "구독 번호가 필요합니다")
		return
	}

	if err := h.subService.CancelSubscription(user.USRSeq, body.SubSeq); err != nil {
		respondError(w, http.StatusBadRequest, "CANCEL_FAILED", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
