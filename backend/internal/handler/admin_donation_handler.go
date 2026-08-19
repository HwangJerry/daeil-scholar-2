// Admin donation handler — HTTP lifecycle for donation config endpoints
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminDonationHandler struct {
	service *service.DonationConfigOrchestrator
}

func NewAdminDonationHandler(svc *service.DonationConfigOrchestrator) *AdminDonationHandler {
	return &AdminDonationHandler{service: svc}
}

func (h *AdminDonationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.GetConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "CONFIG_FAILED", "Failed to get donation config")
		return
	}
	respondJSON(w, http.StatusOK, cfg)
}

type updateDonationConfigRequest struct {
	Goal           int64  `json:"goal"`
	ManualAdj      int64  `json:"manualAdj"`
	ManualDonorCnt int    `json:"manualDonorCnt"`
	Note           string `json:"note"`
	Overwrite      bool   `json:"overwrite"`
}

func (h *AdminDonationHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req updateDonationConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	if err := h.service.UpdateConfig(req.Goal, req.ManualAdj, req.ManualDonorCnt, req.Note, req.Overwrite, user.USRSeq); err != nil {
		respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update config")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminDonationHandler) History(w http.ResponseWriter, r *http.Request) {
	days := parseIntParam(r.URL.Query().Get("days"))
	history, err := h.service.GetHistory(days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "HISTORY_FAILED", "Failed to get history")
		return
	}
	respondJSON(w, http.StatusOK, history)
}

func (h *AdminDonationHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := h.service.ListOrders(model.DonationOrderFilters{
		Name: q.Get("name"), Phone: q.Get("phone"), TransactionNumber: q.Get("transactionNumber"),
		Source: q.Get("source"), Status: q.Get("status"), DonationType: q.Get("donationType"),
	}, parseIntParam(q.Get("page")), parseIntParam(q.Get("size")))
	if err != nil {
		respondDonationOrderError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, page)
}

func (h *AdminDonationHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	seq := parseIntParam(chi.URLParam(r, "orderSeq"))
	order, err := h.service.GetOrder(int64(seq))
	if err != nil {
		respondDonationOrderError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, order)
}

func (h *AdminDonationHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	input, err := decodeDonationOrderInput(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 형식이 올바르지 않습니다.")
		return
	}
	order, err := h.service.CreateOrder(input, user.USRSeq, requestIP(r))
	if err != nil {
		respondDonationOrderError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, order)
}

func (h *AdminDonationHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	seq := parseIntParam(chi.URLParam(r, "orderSeq"))
	input, err := decodeDonationOrderInput(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 형식이 올바르지 않습니다.")
		return
	}
	order, err := h.service.UpdateOrder(int64(seq), input, user.USRSeq, requestIP(r))
	if err != nil {
		respondDonationOrderError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, order)
}

type nullableDonationString struct {
	value *string
	set   bool
}

func (f *nullableDonationString) UnmarshalJSON(data []byte) error {
	f.set = true
	if string(data) == "null" {
		f.value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.value = &value
	return nil
}

type donationDonorRequest struct {
	Name       *string `json:"name"`
	Cohort     *string `json:"cohort"`
	Department *string `json:"department"`
	Phone      *string `json:"phone"`
}

type donationOrderRequest struct {
	Source            *string                `json:"source"`
	AccountUsrSeq     *int                   `json:"accountUsrSeq"`
	TransactionNumber nullableDonationString `json:"transactionNumber"`
	DonationDate      *string                `json:"donationDate"`
	Donor             *donationDonorRequest  `json:"donor"`
	DonationType      *string                `json:"donationType"`
	GrossAmount       *int64                 `json:"grossAmount"`
	RefundedAmount    *int64                 `json:"refundedAmount"`
	Status            *string                `json:"status"`
	PaymentMethod     *string                `json:"paymentMethod"`
	Memo              nullableDonationString `json:"memo"`
}

func decodeDonationOrderInput(r *http.Request) (model.DonationOrderInput, error) {
	var request donationOrderRequest
	if err := decodeClosedJSON(r, &request); err != nil {
		return model.DonationOrderInput{}, err
	}
	if request.Source == nil || !request.TransactionNumber.set || request.DonationDate == nil ||
		request.Donor == nil || request.Donor.Name == nil || request.Donor.Cohort == nil ||
		request.Donor.Department == nil || request.Donor.Phone == nil || request.DonationType == nil ||
		request.GrossAmount == nil || request.RefundedAmount == nil || request.Status == nil ||
		request.PaymentMethod == nil || !request.Memo.set {
		return model.DonationOrderInput{}, errors.New("donation order body requires every canonical field")
	}
	return model.DonationOrderInput{
		Source:            *request.Source,
		AccountUsrSeq:     request.AccountUsrSeq,
		TransactionNumber: request.TransactionNumber.value,
		DonationDate:      *request.DonationDate,
		Donor: model.DonationDonor{
			Name:       *request.Donor.Name,
			Cohort:     *request.Donor.Cohort,
			Department: *request.Donor.Department,
			Phone:      *request.Donor.Phone,
		},
		DonationType:   *request.DonationType,
		GrossAmount:    *request.GrossAmount,
		RefundedAmount: *request.RefundedAmount,
		Status:         *request.Status,
		PaymentMethod:  *request.PaymentMethod,
		Memo:           request.Memo.value,
	}, nil
}

func decodeClosedJSON(r *http.Request, destination interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}

func requestIP(r *http.Request) string {
	if value := r.Header.Get("X-Real-IP"); value != "" {
		return value
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func respondDonationOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrDonationOrderNotFound):
		respondError(w, http.StatusNotFound, "DONATION_ORDER_NOT_FOUND", "기부 거래를 찾을 수 없습니다.")
	case errors.Is(err, repository.ErrDonationOrderConflict):
		respondError(w, http.StatusConflict, "DONATION_ORDER_CONFLICT", "이미 존재하는 기부 거래입니다.")
	case errors.Is(err, service.ErrDonationAccountNotFound):
		respondError(w, http.StatusNotFound, "DONATION_ACCOUNT_NOT_FOUND", "연결할 회원 계정을 찾을 수 없습니다.")
	case errors.Is(err, service.ErrInvalidDonationOrder):
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "요청 값이 올바르지 않습니다.")
	default:
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "요청을 처리하지 못했습니다.")
	}
}
