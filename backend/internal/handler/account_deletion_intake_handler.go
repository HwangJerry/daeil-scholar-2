// account_deletion_intake_handler.go — Root intake of deletion requests received without the app.
package handler

import (
	"encoding/json"
	"github.com/dflh-saf/backend/internal/middleware"
	"io"
	"net/http"
)

type memberSessionRevoker interface {
	RevokeSessions(int) error
}

// CreateOnBehalf registers an emailed request after the operator verified the member's identity.
func (h *AccountDeletionRequestHandler) CreateOnBehalf(w http.ResponseWriter, r *http.Request) {
	operator := middleware.GetAuthUser(r.Context())
	if operator == nil {
		respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	var request struct {
		UserSeq           int    `json:"userSeq"`
		EvidenceReference string `json:"evidenceReference"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&request) != nil {
		respondError(w, 400, "INVALID_REQUEST", "요청 내용을 확인해주세요.")
		return
	}
	if h.RequestsDisabled || (h.TestUserSeq > 0 && request.UserSeq != h.TestUserSeq) {
		respondError(w, http.StatusServiceUnavailable, "ACCOUNT_DELETION_PAUSED", "회원 탈퇴 접수가 일시 중지되어 있습니다. 운영 설정을 먼저 확인해주세요.")
		return
	}
	receipt, receiptToken, cancelToken, err := h.Service.CreateOnBehalf(request.UserSeq, operator.USRSeq, request.EvidenceReference)
	if err != nil {
		deletionRequestError(w, err)
		return
	}
	// Access is already disabled by the committed request. End the member's
	// sessions without clearing the operator's own cookies on this response.
	cleanupPending := true
	if revoker, ok := h.Auth.(memberSessionRevoker); ok {
		cleanupPending = revoker.RevokeSessions(request.UserSeq) != nil
	}
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusCreated, map[string]interface{}{"receipt": receipt, "receiptToken": receiptToken, "cancelToken": cancelToken, "sessionCleanupPending": cleanupPending})
}
