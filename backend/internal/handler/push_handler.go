package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type PushServicer interface {
	RegisterDevice(usrSeq int, registration model.PushDeviceRegistration) error
	UnregisterDevice(usrSeq int, deviceToken string) error
	GetPreferences(usrSeq int) (*model.PushPreferences, error)
	UpdatePreferences(usrSeq int, preferences model.PushPreferences) (*model.PushPreferences, error)
}

type PushHandler struct {
	service PushServicer
}

type pushPreferencesRequest struct {
	MessageEnabled        *bool `json:"messageEnabled"`
	MessagePreviewEnabled *bool `json:"messagePreviewEnabled"`
	NoticeEnabled         *bool `json:"noticeEnabled,omitempty"`
}

func NewPushHandler(service *service.PushService) *PushHandler {
	return &PushHandler{service: service}
}

func (h *PushHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	var request model.PushDeviceRegistration
	if err := decodeClosedPushJSON(r, &request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 본문이 올바르지 않습니다")
		return
	}
	if err := h.service.RegisterDevice(user.USRSeq, request); err != nil {
		h.respondMutationError(w, err, "기기를 등록하지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, model.PushStatusResponse{Status: "registered"})
}

func (h *PushHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	var request model.PushDeviceUnregistration
	if err := decodeClosedPushJSON(r, &request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 본문이 올바르지 않습니다")
		return
	}
	if err := h.service.UnregisterDevice(user.USRSeq, request.DeviceToken); err != nil {
		h.respondMutationError(w, err, "기기 연결을 해제하지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, model.PushStatusResponse{Status: "unregistered"})
}

func (h *PushHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	preferences, err := h.service.GetPreferences(user.USRSeq)
	if err != nil {
		h.respondMutationError(w, err, "알림 설정을 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, preferences)
}

func (h *PushHandler) PutPreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	var request pushPreferencesRequest
	if err := decodeClosedPushJSON(r, &request); err != nil || request.MessageEnabled == nil || request.MessagePreviewEnabled == nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 본문이 올바르지 않습니다")
		return
	}
	preferences, err := h.service.UpdatePreferences(user.USRSeq, model.PushPreferences{
		MessageEnabled:        *request.MessageEnabled,
		MessagePreviewEnabled: *request.MessagePreviewEnabled,
	})
	if err != nil {
		h.respondMutationError(w, err, "알림 설정을 저장하지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, preferences)
}

func (h *PushHandler) respondMutationError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, service.ErrInvalidPushRequest) {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 값이 올바르지 않습니다")
		return
	}
	respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", message)
}

func pushAuthUser(w http.ResponseWriter, r *http.Request) (*model.AuthUser, bool) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return nil, false
	}
	return user, true
}

func decodeClosedPushJSON(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one object")
	}
	return nil
}
