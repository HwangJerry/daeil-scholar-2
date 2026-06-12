package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
)

type PushHandler struct {
	pushService interface {
		RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error
		UnregisterDeviceToken(usrSeq int, token string) error
	}
}

func NewPushHandler(pushService interface {
	RegisterDeviceToken(usrSeq int, req model.PushDeviceRegistrationRequest) error
	UnregisterDeviceToken(usrSeq int, token string) error
}) *PushHandler {
	return &PushHandler{pushService: pushService}
}

type registerPushDeviceRequest struct {
	Platform    string `json:"platform"`
	DeviceToken string `json:"deviceToken"`
	Locale      string `json:"locale"`
}

func (h *PushHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var req registerPushDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문이 올바르지 않습니다")
		return
	}
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	if req.DeviceToken = strings.TrimSpace(req.DeviceToken); req.DeviceToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "디바이스 토큰이 필요합니다")
		return
	}
	if req.Platform != "ios" {
		respondError(w, http.StatusBadRequest, "INVALID_PLATFORM", "지원하지 않는 플랫폼입니다")
		return
	}

	payload := model.PushDeviceRegistrationRequest{
		Platform:    req.Platform,
		DeviceToken: req.DeviceToken,
		Locale:      strings.TrimSpace(req.Locale),
	}

	if err := h.pushService.RegisterDeviceToken(user.USRSeq, payload); err != nil {
		respondError(w, http.StatusInternalServerError, "PUSH_REGISTER_FAILED", "디바이스 등록에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusCreated, model.PushDeviceRegistrationResponse{Status: "ok"})
}

type unregisterPushDeviceRequest struct {
	DeviceToken string `json:"deviceToken"`
}

func (h *PushHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	var req unregisterPushDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "요청 본문이 올바르지 않습니다")
		return
	}
	token := strings.TrimSpace(req.DeviceToken)
	if token == "" {
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "디바이스 토큰이 필요합니다")
		return
	}
	if err := h.pushService.UnregisterDeviceToken(user.USRSeq, token); err != nil {
		respondError(w, http.StatusInternalServerError, "PUSH_UNREGISTER_FAILED", "디바이스 해제에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
