package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
)

func TestAdminDonationCreateOrderRejectsUnknownField(t *testing.T) {
	handler := NewAdminDonationHandler(nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/donation/orders", strings.NewReader(`{
		"source":"other",
		"transactionNumber":null,
		"donationDate":"2026-07-28",
		"donor":{"name":"기부자","cohort":"18","department":"영어","phone":"01000000000"},
		"donationType":"one_time",
		"grossAmount":100000,
		"refundedAmount":0,
		"status":"completed",
		"paymentMethod":"admin",
		"memo":null,
		"unexpected":true
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	handler.CreateOrder(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
		t.Fatalf("body = %s, want INVALID_REQUEST", response.Body.String())
	}
}

func TestAdminDonationCreateOrderRejectsMissingFullReplacementField(t *testing.T) {
	handler := NewAdminDonationHandler(nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/donation/orders", strings.NewReader(`{
		"source":"other",
		"transactionNumber":null,
		"donationDate":"2026-07-28",
		"donor":{"name":"기부자","cohort":"18","department":"영어","phone":"01000000000"},
		"donationType":"one_time",
		"refundedAmount":0,
		"status":"completed",
		"paymentMethod":"admin",
		"memo":null
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	handler.CreateOrder(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
		t.Fatalf("body = %s, want INVALID_REQUEST", response.Body.String())
	}
}
