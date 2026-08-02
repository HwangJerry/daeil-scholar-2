package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type MemberBlockServicer interface {
	List(blockerSeq int) (*model.MemberBlockListResponse, error)
	Get(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
	Block(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
	Unblock(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
}

type MemberBlockHandler struct {
	service MemberBlockServicer
}

func NewMemberBlockHandler(service *service.MemberBlockService) *MemberBlockHandler {
	return &MemberBlockHandler{service: service}
}

func (h *MemberBlockHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := blockAuthUser(w, r)
	if !ok {
		return
	}
	result, err := h.service.List(user.USRSeq)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "차단 목록을 불러오지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *MemberBlockHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.respondState(w, r, h.service.Get)
}

func (h *MemberBlockHandler) Put(w http.ResponseWriter, r *http.Request) {
	h.respondState(w, r, h.service.Block)
}

func (h *MemberBlockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.respondState(w, r, h.service.Unblock)
}

func (h *MemberBlockHandler) respondState(w http.ResponseWriter, r *http.Request, action func(int, int) (*model.MemberBlockState, error)) {
	user, ok := blockAuthUser(w, r)
	if !ok {
		return
	}
	targetSeq, err := strconv.Atoi(chi.URLParam(r, "userSeq"))
	if err != nil || targetSeq <= 0 {
		respondError(w, http.StatusBadRequest, "INVALID_USER_SEQ", "회원 식별자가 올바르지 않습니다")
		return
	}
	state, err := action(user.USRSeq, targetSeq)
	if err != nil {
		if errors.Is(err, service.ErrMemberBlockTargetNotFound) {
			respondError(w, http.StatusNotFound, "INVALID_USER_SEQ", "동문 정보를 찾을 수 없습니다")
			return
		}
		var validationErr *model.ValidationError
		if errors.As(err, &validationErr) {
			respondError(w, http.StatusBadRequest, "INVALID_USER_SEQ", validationErr.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "INVALID_REQUEST", "차단 상태를 변경하지 못했습니다")
		return
	}
	respondJSON(w, http.StatusOK, state)
}

func blockAuthUser(w http.ResponseWriter, r *http.Request) (*model.AuthUser, bool) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return nil, false
	}
	return user, true
}
