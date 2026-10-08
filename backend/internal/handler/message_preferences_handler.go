// message_preferences_handler.go — Authenticated receiving preference API.
package handler

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
)

type MessagePreferencesHandler struct {
	Service *service.MessagePreferencesService
}

func (h *MessagePreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	preferences, err := h.Service.Get(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "설정을 불러오지 못했어요.")
		return
	}
	respondJSON(w, http.StatusOK, preferences)
}

func (h *MessagePreferencesHandler) Put(w http.ResponseWriter, r *http.Request) {
	user, ok := pushAuthUser(w, r)
	if !ok {
		return
	}
	var request struct {
		MessageAllowed *bool `json:"messageAllowed"`
	}
	if err := decodeClosedPushJSON(r, &request); err != nil || request.MessageAllowed == nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 본문이 올바르지 않습니다")
		return
	}
	preferences, err := h.Service.Save(user.USRSeq, model.MessagePreferences{MessageAllowed: *request.MessageAllowed})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "설정을 저장하지 못했어요.")
		return
	}
	respondJSON(w, http.StatusOK, preferences)
}
