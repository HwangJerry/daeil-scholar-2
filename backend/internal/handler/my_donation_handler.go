// my_donation_handler.go — HTTP handler for user donation history
package handler

import (
	"net/http"

	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
)

type MyDonationHandler struct {
	service *service.MyDonationService
}

func NewMyDonationHandler(svc *service.MyDonationService) *MyDonationHandler {
	return &MyDonationHandler{service: svc}
}

func (h *MyDonationHandler) GetMyDonations(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	q := r.URL.Query()
	sort := q.Get("sort")
	page := parseIntParam(q.Get("page"))
	size := parseIntParam(q.Get("size"))

	result, err := h.service.GetMyDonations(user.USRSeq, sort, page, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "기부내역 조회에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
