package handler

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/dflh-saf/backend/internal/config"
	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/easypay_relay.html
var relayTemplateContent string

type PaymentHandler struct {
	donateService *service.DonateService
	easypayConfig config.EasyPayConfig
	relayTmpl     *template.Template
}

func NewPaymentHandler(donateService *service.DonateService, cfg config.EasyPayConfig) *PaymentHandler {
	tmpl := template.Must(template.New("relay").Parse(relayTemplateContent))
	return &PaymentHandler{donateService: donateService, easypayConfig: cfg, relayTmpl: tmpl}
}

func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
		return
	}

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}

	result, err := h.donateService.CreateOrder(user, req, ip, h.easypayConfig)
	if err != nil {
		respondError(w, http.StatusBadRequest, "ORDER_FAILED", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *PaymentHandler) EasyPayRelay(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.relayTmpl.Execute(w, r.Form)
}

func (h *PaymentHandler) EasyPayReturn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/donation/result?status=failed&reason=bad_request", http.StatusFound)
		return
	}

	resCode := r.FormValue("sp_res_cd")
	orderNo := r.FormValue("sp_order_no")

	if resCode != "0000" {
		http.Redirect(w, r, "/donation/result?status=failed&reason=pg_error", http.StatusFound)
		return
	}

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}

	err := h.donateService.ConfirmPayment(
		orderNo,
		r.FormValue("sp_encrypt_data"),
		r.FormValue("sp_sessionkey"),
		r.FormValue("sp_trace_no"),
		ip,
	)
	if err != nil {
		http.Redirect(w, r, "/donation/result?status=failed&reason=server_error", http.StatusFound)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/donation/result?status=success&order=%s", orderNo), http.StatusFound)
}

func (h *PaymentHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	seq := parseIntParam(chi.URLParam(r, "seq"))
	if seq == 0 {
		respondError(w, http.StatusBadRequest, "INVALID_PARAM", "주문 번호가 필요합니다")
		return
	}

	order, err := h.donateService.GetOrder(seq, user.USRSeq)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "주문을 찾을 수 없습니다")
		return
	}
	respondJSON(w, http.StatusOK, order)
}
