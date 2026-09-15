// admin_app_update_handler.go — Administrator endpoints for per-platform app update policies.
package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AppUpdatePolicyServicer interface {
	ListPolicies() ([]model.AppUpdatePolicyRecord, error)
	SavePolicy(input service.SaveAppUpdatePolicyInput) error
	ListHistory(platform string) ([]model.AppUpdatePolicyHistoryEntry, error)
}

type AppClientBuildServicer interface {
	List(platform string) ([]model.AppClientBuild, error)
}

type AdminAppUpdateHandler struct {
	policies AppUpdatePolicyServicer
	builds   AppClientBuildServicer
}

func NewAdminAppUpdateHandler(policies AppUpdatePolicyServicer, builds AppClientBuildServicer) *AdminAppUpdateHandler {
	return &AdminAppUpdateHandler{policies: policies, builds: builds}
}

type appUpdatePolicyListResponse struct {
	Policies []model.AppUpdatePolicyRecord `json:"policies"`
}

func (h *AdminAppUpdateHandler) ListPolicies(w http.ResponseWriter, _ *http.Request) {
	policies, err := h.policies.ListPolicies()
	if err != nil {
		respondAppUpdateError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, appUpdatePolicyListResponse{Policies: policies})
}

type saveAppUpdatePolicyRequest struct {
	Policy *model.AppUpdatePolicy `json:"policy"`
	// UpdatedAt is the value last read by the administrator. It makes concurrent
	// edits fail loudly instead of silently overwriting each other.
	UpdatedAt *time.Time `json:"updatedAt"`
	// ExpectedPolicy is the policy the administrator was editing. It catches a
	// concurrent save that lands within the same second as their own read.
	ExpectedPolicy *model.AppUpdatePolicy `json:"expectedPolicy"`
	// AllowUnobservedBuild confirms a threshold the API has never seen from this
	// platform, which the administrator must opt into explicitly.
	AllowUnobservedBuild bool `json:"allowUnobservedBuild"`
}

func (h *AdminAppUpdateHandler) SavePolicy(w http.ResponseWriter, r *http.Request) {
	var request saveAppUpdatePolicyRequest
	if err := decodeClosedJSON(r, &request); err != nil || request.Policy == nil ||
		request.UpdatedAt == nil || request.UpdatedAt.IsZero() {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "policy와 updatedAt이 필요합니다")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	err := h.policies.SavePolicy(service.SaveAppUpdatePolicyInput{
		Platform:             chi.URLParam(r, "platform"),
		Policy:               *request.Policy,
		ExpectedUpdatedAt:    *request.UpdatedAt,
		ExpectedPolicy:       request.ExpectedPolicy,
		AllowUnobservedBuild: request.AllowUnobservedBuild,
		UpdatedBy:            user.USRSeq,
	})
	if err != nil {
		respondAppUpdateError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type appUpdatePolicyHistoryResponse struct {
	Entries []model.AppUpdatePolicyHistoryEntry `json:"entries"`
}

func (h *AdminAppUpdateHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	entries, err := h.policies.ListHistory(chi.URLParam(r, "platform"))
	if err != nil {
		respondAppUpdateError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, appUpdatePolicyHistoryResponse{Entries: entries})
}

type appClientBuildListResponse struct {
	Builds []model.AppClientBuild `json:"builds"`
}

func (h *AdminAppUpdateHandler) ListClientBuilds(w http.ResponseWriter, r *http.Request) {
	builds, err := h.builds.List(r.URL.Query().Get("platform"))
	if err != nil {
		respondAppUpdateError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, appClientBuildListResponse{Builds: builds})
}

// respondAppUpdateError maps policy errors onto the status codes the admin SPA
// reacts to: field-level 400, conflict 409, missing policy 404.
func respondAppUpdateError(w http.ResponseWriter, err error) {
	var validation *service.AppUpdatePolicyValidationError
	switch {
	case errors.As(err, &validation):
		respondJSON(w, http.StatusBadRequest, model.APIError{
			Code:    "INVALID_POLICY",
			Message: "정책 값을 저장할 수 없습니다",
			Details: map[string][]service.AppUpdatePolicyFieldError{"fields": validation.Fields},
		})
	case errors.Is(err, service.ErrUnsupportedAppUpdatePlatform):
		respondError(w, http.StatusBadRequest, "INVALID_PLATFORM", "지원하지 않는 플랫폼입니다")
	case errors.Is(err, service.ErrAppUpdatePolicyConflict):
		respondError(w, http.StatusConflict, "POLICY_CONFLICT", "다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요")
	case errors.Is(err, service.ErrAppUpdatePolicyNotFound):
		respondError(w, http.StatusNotFound, "POLICY_NOT_FOUND", "앱 업데이트 정책을 찾을 수 없습니다")
	default:
		respondError(w, http.StatusInternalServerError, "POLICY_FAILED", "앱 업데이트 정책 처리에 실패했습니다")
	}
}
