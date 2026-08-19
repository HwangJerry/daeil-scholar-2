package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func TestAdminDonationUpdateConfigReturnsBadRequestForInvalidTierThresholds(t *testing.T) {
	adminService := service.NewAdminDonationService(nil, nil)
	orchestrator := service.NewDonationConfigOrchestrator(adminService, nil, nil)
	handler := NewAdminDonationHandler(orchestrator)
	request := httptest.NewRequest(http.MethodPut, "/api/admin/donation/config", strings.NewReader(`{
		"goal":200000000,
		"manualAdj":0,
		"manualDonorCnt":0,
		"tierSproutMin":1,
		"tierSaplingMin":10000,
		"tierTreeMin":50000,
		"tierBloomingMin":50000,
		"tierFruitingMin":300000,
		"note":"invalid thresholds",
		"overwrite":false
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	handler.UpdateConfig(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_TIER_THRESHOLDS"`) {
		t.Fatalf("body = %s, want INVALID_TIER_THRESHOLDS", response.Body.String())
	}
}
