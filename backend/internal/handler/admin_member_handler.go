// Admin member handler — HTTP lifecycle for member management endpoints
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminMemberHandler struct {
	service *service.AdminMemberService
}

func NewAdminMemberHandler(svc *service.AdminMemberService) *AdminMemberHandler {
	return &AdminMemberHandler{service: svc}
}

func (h *AdminMemberHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, total, err := h.service.List(
		parseIntParam(q.Get("page")),
		parseIntParam(q.Get("size")),
		q.Get("q"), q.Get("fn"), q.Get("status"),
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LIST_FAILED", "Failed to list members")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": rows, "total": total})
}

func (h *AdminMemberHandler) Detail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	detail, err := h.service.GetDetail(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DETAIL_FAILED", "Failed to load member")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Member not found")
		return
	}
	hasKakao, _ := h.service.HasKakaoLink(seq)
	respondJSON(w, http.StatusOK, map[string]interface{}{"member": detail, "kakaoLinked": hasKakao})
}

func (h *AdminMemberHandler) ListAlumniVerifications(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListAlumniVerifications(r.URL.Query().Get("status"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "VERIFICATION_LIST_FAILED", "동문 인증 목록을 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *AdminMemberHandler) GetAlumniVerificationDetail(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "userSeq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid user sequence")
		return
	}
	detail, err := h.service.GetAlumniVerificationDetail(seq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "VERIFICATION_DETAIL_FAILED", "동문 인증 정보를 불러오지 못했습니다")
		return
	}
	if detail == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "동문 인증 정보를 찾을 수 없습니다")
		return
	}
	respondJSON(w, http.StatusOK, detail)
}

type updateMemberRequest struct {
	Status string `json:"status"`
}

type rejectAlumniVerificationRequest struct {
	Reason            string    `json:"reason"`
	ExpectedUpdatedAt time.Time `json:"expectedUpdatedAt"`
}

type approveAlumniVerificationRequest struct {
	ExpectedUpdatedAt time.Time `json:"expectedUpdatedAt"`
}

func (h *AdminMemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid seq")
		return
	}
	var req updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.UpdateStatus(seq, req.Status); err != nil {
		if errors.Is(err, service.ErrLegacyVerificationStatusNotAllowed) {
			respondError(w, http.StatusConflict, "VERIFICATION_STATE_CONFLICT", "동문 인증 상태는 전용 심사 API로 변경해야 합니다")
			return
		}
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminMemberHandler) RejectAlumniVerification(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "userSeq"))
	admin := middleware.GetAuthUser(r.Context())
	if seq <= 0 || admin == nil {
		respondError(w, http.StatusBadRequest, "INVALID_REVIEW_REQUEST", "Invalid review request")
		return
	}
	var req rejectAlumniVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.ExpectedUpdatedAt.IsZero() {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "expectedUpdatedAt is required")
		return
	}
	if err := h.service.RejectAlumniVerification(seq, admin.USRSeq, req.Reason, req.ExpectedUpdatedAt); err != nil {
		writeVerificationReviewError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminMemberHandler) ApproveAlumniVerification(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "userSeq"))
	admin := middleware.GetAuthUser(r.Context())
	if seq <= 0 || admin == nil {
		respondError(w, http.StatusBadRequest, "INVALID_REVIEW_REQUEST", "Invalid review request")
		return
	}
	var req approveAlumniVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.ExpectedUpdatedAt.IsZero() {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "expectedUpdatedAt is required")
		return
	}
	if err := h.service.ApproveAlumniVerification(seq, admin.USRSeq, req.ExpectedUpdatedAt); err != nil {
		writeVerificationReviewError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeVerificationReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrRejectionReasonRequired):
		respondError(w, http.StatusBadRequest, "REJECTION_REASON_REQUIRED", "반려 사유를 입력해주세요")
	case errors.Is(err, service.ErrVerificationStale):
		respondError(w, http.StatusConflict, "VERIFICATION_STALE", "동문 인증 정보가 변경되었습니다")
	case errors.Is(err, service.ErrVerificationStateConflict):
		respondError(w, http.StatusConflict, "VERIFICATION_STATE_CONFLICT", "현재 상태에서는 심사할 수 없습니다")
	default:
		respondError(w, http.StatusInternalServerError, "VERIFICATION_REVIEW_FAILED", "동문 인증 심사에 실패했습니다")
	}
}

func (h *AdminMemberHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetMemberStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STATS_FAILED", "Failed to get member stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}
