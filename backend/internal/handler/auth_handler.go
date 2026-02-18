package handler

import (
	"encoding/json"
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
	user, err := h.service.FindMemberByKakaoID(kakaoID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "Failed to lookup member")
		return
	}
	if user != nil {
		if err := h.service.LoginWithBridge(user, w, r); err != nil {
			respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "Failed to login")
			return
		}
		h.service.CacheKakaoToken(user.USRSeq, accessToken)
		http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/", http.StatusFound)
		return
	}
	// Store Kakao info server-side with a temporary token (not in URL)
	linkToken := h.service.GenerateSessionID()
	h.cache.Set("kakao_link:"+linkToken, model.KakaoLinkData{
		KakaoID:     kakaoID,
		Email:       email,
		Nickname:    nickname,
		AccessToken: accessToken,
	}, 5*time.Minute)
	http.Redirect(w, r, h.cfg.Server.AllowedOrigin+"/login/link?token="+linkToken, http.StatusFound)
}

type kakaoLinkRequest struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	FN    string `json:"fn"`
}

func (h *AuthHandler) KakaoLink(w http.ResponseWriter, r *http.Request) {
	var req kakaoLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Token == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing required fields")
		return
	}
	// Retrieve Kakao info from server-side cache using the link token
	cached, found := h.cache.Get("kakao_link:" + req.Token)
	if !found {
		respondError(w, http.StatusBadRequest, "INVALID_TOKEN", "Link token expired or invalid")
		return
	}
	linkData, ok := cached.(model.KakaoLinkData)
	if !ok {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid cached data")
		return
	}
	h.cache.Delete("kakao_link:" + req.Token)

	if req.Phone == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "전화번호를 입력하세요")
		return
	}

	// Look up existing member by phone number
	existing, err := h.memberSvc.FindMemberByPhone(req.Phone)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "회원 조회 중 오류가 발생했습니다")
		return
	}

	var user *model.User
	if existing != nil {
		// Phone matched — verify name matches for merge
		if existing.USRName != req.Name {
			respondError(w, http.StatusConflict, "NAME_MISMATCH", "전화번호는 등록되어 있으나 이름이 일치하지 않습니다")
			return
		}
		// Merge: link Kakao account to existing member
		if err := h.service.InsertSocialLink(existing.USRSeq, "KT", linkData.KakaoID, req.Name); err != nil {
			respondError(w, http.StatusInternalServerError, "LINK_FAILED", "계정 연동에 실패했습니다")
			return
		}
		user = existing
	} else {
		// No existing member — create new member and link Kakao
		newUser, err := h.memberSvc.CreateMember(linkData.KakaoID, req.Name, req.Phone, req.FN, linkData.Email)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "CREATE_FAILED", "회원 가입에 실패했습니다")
			return
		}
		if err := h.service.InsertSocialLink(newUser.USRSeq, "KT", linkData.KakaoID, req.Name); err != nil {
			respondError(w, http.StatusInternalServerError, "LINK_FAILED", "계정 연동에 실패했습니다")
			return
		}
		user = newUser
	}

	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	h.service.CacheKakaoToken(user.USRSeq, linkData.AccessToken)
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	respondJSON(w, http.StatusOK, authUser)
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
