// admin_operator_handler.go — Root-only HTTP endpoints for listing, granting and revoking admin roles
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminOperatorHandler struct {
	service *service.AdminOperatorService
}

func NewAdminOperatorHandler(svc *service.AdminOperatorService) *AdminOperatorHandler {
	return &AdminOperatorHandler{service: svc}
}

type setOperatorRoleRequest struct {
	Role model.AdminRole `json:"role"`
}

// List handles GET /api/admin/operators.
func (h *AdminOperatorHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "OPERATOR_LIST_FAILED", "운영자 목록을 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// SetRole handles PUT /api/admin/operators/{usrSeq}.
func (h *AdminOperatorHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	actor, target, ok := operatorRequestParties(w, r)
	if !ok {
		return
	}
	var req setOperatorRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.service.SetRole(actor, target, req.Role); err != nil {
		writeOperatorError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Revoke handles DELETE /api/admin/operators/{usrSeq}.
func (h *AdminOperatorHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	actor, target, ok := operatorRequestParties(w, r)
	if !ok {
		return
	}
	if err := h.service.Revoke(actor, target); err != nil {
		writeOperatorError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func operatorRequestParties(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	admin := middleware.GetAuthUser(r.Context())
	target := parseIntParam(chi.URLParam(r, "usrSeq"))
	if admin == nil || target <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_SEQ", "Invalid user sequence")
		return 0, 0, false
	}
	return admin.USRSeq, target, true
}

func writeOperatorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAdminRole):
		respondError(w, http.StatusBadRequest, "INVALID_ROLE", "권한은 root 또는 operator만 지정할 수 있습니다")
	case errors.Is(err, service.ErrSelfAdminRoleChange):
		respondError(w, http.StatusConflict, "SELF_ROLE_CHANGE", "본인의 권한은 변경할 수 없습니다")
	case errors.Is(err, service.ErrLastRootAdmin):
		respondError(w, http.StatusConflict, "LAST_ROOT_ADMIN", "root 관리자는 최소 1명 있어야 합니다")
	case errors.Is(err, service.ErrOperatorTargetIneligible):
		respondError(w, http.StatusConflict, "TARGET_INELIGIBLE", "탈퇴·휴면·정지 회원이나 없는 회원은 운영자로 지정할 수 없습니다")
	case errors.Is(err, service.ErrOperatorNotFound):
		respondError(w, http.StatusNotFound, "NOT_FOUND", "관리자 권한이 없는 회원입니다")
	default:
		respondError(w, http.StatusInternalServerError, "OPERATOR_UPDATE_FAILED", "운영자 권한 변경에 실패했습니다")
	}
}
