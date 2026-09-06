// account_deletion_request_handler.go — In-app requests, private receipts and administrator processing.
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

type AccountDeletionSessionService interface {
	LogoutAll(http.ResponseWriter, int) error
}

type AccountDeletionRequestHandler struct {
	Service *service.AccountDeletionRequestService
	Auth    AccountDeletionSessionService
}

func deletionRequestError(w http.ResponseWriter, err error) {
	var invalid *model.ValidationError
	switch {
	case errors.As(err, &invalid):
		respondError(w, 400, "INVALID_DELETION_REQUEST", invalid.Error())
	case errors.Is(err, sql.ErrNoRows):
		respondError(w, 404, "DELETION_REQUEST_NOT_FOUND", "삭제 요청을 찾을 수 없거나 이미 처리되었습니다.")
	case errors.Is(err, repository.ErrDeletionReceiptConflict):
		respondError(w, 409, "DELETION_ALREADY_REQUESTED", "이미 접수된 삭제 요청입니다. 기존 확인번호로 조회해주세요.")
	case errors.Is(err, repository.ErrDeletionIncomplete):
		respondError(w, 409, "DELETION_INCOMPLETE", "계정 관련 기록 또는 소셜 권한 철회 확인이 남아 있습니다. 실제 삭제 후 다시 확인해주세요.")
	default:
		respondError(w, 500, "DELETION_REQUEST_FAILED", "삭제 요청을 처리하지 못했습니다. 다시 시도해주세요.")
	}
}

func (h *AccountDeletionRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	var request struct {
		ReceiptToken string `json:"receiptToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		respondError(w, 400, "INVALID_DELETION_REQUEST", "요청 내용을 확인해주세요.")
		return
	}
	receipt, token, err := h.Service.Create(user.USRSeq, request.ReceiptToken)
	if err != nil {
		deletionRequestError(w, err)
		return
	}
	// Receipt and access disablement have already committed. Even when remote
	// session cleanup fails, never invite a second request by returning 500.
	cleanupPending := h.Auth.LogoutAll(w, user.USRSeq) != nil
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusAccepted, map[string]interface{}{"receipt": receipt, "receiptToken": token, "sessionCleanupPending": cleanupPending})
}

func (h *AccountDeletionRequestHandler) Receipt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var request struct {
		ReceiptToken string `json:"receiptToken"`
	}
	if json.NewDecoder(r.Body).Decode(&request) != nil {
		respondError(w, 400, "INVALID_RECEIPT", "확인번호를 입력해주세요.")
		return
	}
	receipt, err := h.Service.Receipt(request.ReceiptToken)
	if err != nil {
		deletionRequestError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, receipt)
}

func (h *AccountDeletionRequestHandler) List(w http.ResponseWriter, r *http.Request) {
	before := int64(0)
	if raw := r.URL.Query().Get("before"); raw != "" {
		var err error
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			respondError(w, 400, "INVALID_CURSOR", "잘못된 페이지입니다.")
			return
		}
	}
	items, err := h.Service.List(r.URL.Query().Get("status"), before)
	if err != nil {
		deletionRequestError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *AccountDeletionRequestHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, 400, "INVALID_REQUEST_ID", "잘못된 요청입니다.")
		return
	}
	items, err := h.Service.Verify(id)
	if err != nil {
		deletionRequestError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *AccountDeletionRequestHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다.")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var request model.AccountDeletionResolution
	if err != nil || json.NewDecoder(r.Body).Decode(&request) != nil {
		respondError(w, 400, "INVALID_REQUEST", "요청 내용을 확인해주세요.")
		return
	}
	if err = h.Service.Resolve(id, user.USRSeq, request); err != nil {
		deletionRequestError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
