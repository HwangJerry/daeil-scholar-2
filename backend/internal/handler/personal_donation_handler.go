package handler

import (
	"errors"
	"net/http"

	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
)

type PersonalDonationHandler struct {
	service *service.PersonalDonationService
}

func NewPersonalDonationHandler(personalDonationService *service.PersonalDonationService) *PersonalDonationHandler {
	return &PersonalDonationHandler{service: personalDonationService}
}

func (h *PersonalDonationHandler) GetMyDonations(w http.ResponseWriter, r *http.Request) {
	user := mw.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}

	query := r.URL.Query()
	result, err := h.service.GetPersonalDonations(
		user.USRSeq,
		query.Get("sort"),
		parseIntParam(query.Get("page")),
		parseIntParam(query.Get("size")),
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPersonalDonationUser) {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
			return
		}
		respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "기부내역 조회에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, result)
}
