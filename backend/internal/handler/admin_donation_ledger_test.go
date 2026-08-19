package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
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

func TestDecodeDonationOrderInputDistinguishesAccountLinkStates(t *testing.T) {
	tests := []struct {
		name      string
		account   string
		wantSet   bool
		wantValue *int
	}{
		{name: "omitted", wantSet: false},
		{name: "null", account: `,"accountUsrSeq":null`, wantSet: true},
		{name: "value", account: `,"accountUsrSeq":42`, wantSet: true, wantValue: intPointer(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{
				"source":"other"` + tt.account + `,
				"transactionNumber":null,
				"donationDate":"2026-07-28",
				"donor":{"name":"기부자","cohort":"18","department":"영어","phone":"01000000000"},
				"donationType":"one_time",
				"grossAmount":100000,
				"refundedAmount":0,
				"status":"completed",
				"paymentMethod":"admin",
				"memo":null,
				"lastEditedAt":"2026-08-20T12:00:00Z"
			}`
			request := httptest.NewRequest(http.MethodPut, "/api/admin/donation/orders/3001", strings.NewReader(body))

			input, err := decodeDonationOrderInput(request)
			if err != nil {
				t.Fatalf("decodeDonationOrderInput() error = %v", err)
			}
			if input.AccountUsrSeqSet != tt.wantSet {
				t.Fatalf("AccountUsrSeqSet = %v, want %v", input.AccountUsrSeqSet, tt.wantSet)
			}
			if tt.wantValue == nil {
				if input.AccountUsrSeq != nil {
					t.Fatalf("AccountUsrSeq = %v, want nil", *input.AccountUsrSeq)
				}
				return
			}
			if input.AccountUsrSeq == nil || *input.AccountUsrSeq != *tt.wantValue {
				t.Fatalf("AccountUsrSeq = %v, want %d", input.AccountUsrSeq, *tt.wantValue)
			}
		})
	}
}

func TestAdminDonationUpdateRequiresLastEditedAt(t *testing.T) {
	handler := NewAdminDonationHandler(nil)
	request := httptest.NewRequest(http.MethodPut, "/api/admin/donation/orders/3001", strings.NewReader(`{
		"source":"other",
		"transactionNumber":null,
		"donationDate":"2026-07-28",
		"donor":{"name":"기부자","cohort":"18","department":"영어","phone":"01000000000"},
		"donationType":"one_time",
		"grossAmount":100000,
		"refundedAmount":0,
		"status":"completed",
		"paymentMethod":"admin",
		"memo":null
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	handler.UpdateOrder(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_REQUEST"`) {
		t.Fatalf("response = %d %s, want 400 INVALID_REQUEST", response.Code, response.Body.String())
	}
}

func TestAdminDonationStaleUpdateReturnsConflict(t *testing.T) {
	response := httptest.NewRecorder()

	respondDonationOrderError(response, repository.ErrDonationOrderStale)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"code":"DONATION_ORDER_STALE"`) ||
		!strings.Contains(body, "다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해주세요.") {
		t.Fatalf("body = %s", body)
	}
}

func intPointer(value int) *int {
	return &value
}
