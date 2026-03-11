package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type AuthHandler struct {
	service   *service.AuthService
	memberSvc *service.MemberService
	cache     *cache.Cache
	cfg       *config.Config
	logger    zerolog.Logger
}

func NewAuthHandler(service *service.AuthService, memberSvc *service.MemberService, cacheStore *cache.Cache, cfg *config.Config, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{service: service, memberSvc: memberSvc, cache: cacheStore, cfg: cfg, logger: logger}
}

func (h *AuthHandler) KakaoLogin(w http.ResponseWriter, r *http.Request) {
	state := h.service.GenerateSessionID()
	if state == "" {
		respondError(w, http.StatusInternalServerError, "STATE_FAILED", "Failed to generate state")
		return
	}
	h.cache.Set("oauth_state:"+state, true, 5*time.Minute)
	authURL := url.URL{
		Scheme: "https",
		Host:   "kauth.kakao.com",
		Path:   "/oauth/authorize",
	}
	query := authURL.Query()
	query.Set("client_id", h.cfg.Kakao.ClientID)
	query.Set("redirect_uri", h.cfg.Kakao.RedirectURI)
	query.Set("response_type", "code")
	query.Set("state", state)
	authURL.RawQuery = query.Encode()
	h.logger.Debug().
		Str("client_id", h.cfg.Kakao.ClientID).
		Str("redirect_uri", h.cfg.Kakao.RedirectURI).
		Str("auth_url", authURL.String()).
		Msg("kakao: redirecting to authorize")
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (h *AuthHandler) KakaoCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if _, found := h.cache.Get("oauth_state:" + state); !found {
		respondError(w, http.StatusForbidden, "INVALID_STATE", "OAuth state validation failed")
		return
	}
	h.cache.Delete("oauth_state:" + state)
	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "INVALID_CODE", "Missing code")
		return
	}
	h.logger.Debug().
		Str("code", code[:min(len(code), 8)]+"...").
		Str("state", state).
		Msg("kakao: callback received")
	kakaoID, email, nickname, accessToken, err := h.service.ExchangeKakaoToken(code)
	if err != nil {
		h.logger.Error().Err(err).Msg("kakao: token exchange failed")
		respondError(w, http.StatusBadRequest, "KAKAO_EXCHANGE_FAILED", "Kakao token exchange failed")
		return
	}
	h.logger.Debug().
		Str("kakao_id", kakaoID).
		Str("email", email).
		Str("nickname", nickname).
		Msg("kakao: token exchanged")
	h.handleSocialCallback(w, r, "KT", kakaoID, email, nickname, accessToken)
}

// KakaoLink delegates to SocialLink for backward compatibility.
func (h *AuthHandler) KakaoLink(w http.ResponseWriter, r *http.Request) {
	h.SocialLink(w, r)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.USRID == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "아이디와 비밀번호를 입력하세요")
		return
	}
	user, err := h.memberSvc.LoginWithPassword(req.USRID, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrPendingApproval) {
			respondError(w, http.StatusForbidden, "PENDING_APPROVAL", "가입 신청이 접수된 계정입니다. 관리자 승인 후 로그인 가능합니다.")
			return
		}
		h.logger.Error().Err(err).Str("usrId", req.USRID).Msg("login: password verification failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	if user == nil {
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "아이디 또는 비밀번호가 올바르지 않습니다")
		return
	}
	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("login: bridge session failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	respondJSON(w, http.StatusOK, authUser)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.UsrID == "" || req.Password == "" || req.Name == "" || req.Phone == "" || req.Email == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "필수 입력값이 누락되었습니다")
		return
	}
	if len(req.UsrID) < 4 || len(req.UsrID) > 20 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "아이디는 4~20자여야 합니다")
		return
	}
	user, err := h.memberSvc.RegisterMember(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIDTaken):
			respondError(w, http.StatusConflict, "ID_TAKEN", "이미 사용 중인 아이디입니다")
		case errors.Is(err, service.ErrPhoneTaken):
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 등록된 전화번호입니다")
		default:
			h.logger.Error().Err(err).Msg("register: failed to create member")
			respondError(w, http.StatusInternalServerError, "REGISTER_FAILED", "회원가입 처리 중 오류가 발생했습니다")
		}
		return
	}
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	respondJSON(w, http.StatusCreated, authUser)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	usrSeq := 0
	if user != nil {
		usrSeq = user.USRSeq
	}
	h.service.Logout(w, usrSeq)
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	respondJSON(w, http.StatusOK, user)
}
