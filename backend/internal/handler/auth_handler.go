package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

var fnDigitRegex = regexp.MustCompile(`^[0-9]+$`)

type AuthHandler struct {
	service          *service.AuthService
	mobileIssuer     *service.MobileSessionIssuer
	socialAuth       *service.SocialAuthService
	appleVerifier    *service.AppleIdentityVerifier
	socialLifecycle  *service.SocialAccountLifecycleService
	memberSvc        *service.MemberService
	registerSvc      *service.RegistrationService
	profileSvc       *service.ProfileService
	cache            *cache.Cache
	socialLinkTokens *service.SocialLinkTokenStore
	cfg              *config.Config
	logger           zerolog.Logger
}

func NewAuthHandler(
	svc *service.AuthService,
	memberSvc *service.MemberService,
	registerSvc *service.RegistrationService,
	profileSvc *service.ProfileService,
	cacheStore *cache.Cache,
	socialLinkTokens *service.SocialLinkTokenStore,
	cfg *config.Config,
	logger zerolog.Logger,
) *AuthHandler {
	mobileIssuer := service.NewMobileSessionIssuer(svc)
	appleVerifier := service.NewAppleIdentityVerifier(svc, cfg.Apple)
	socialLifecycle := service.NewSocialAccountLifecycleService(svc, appleVerifier)
	return &AuthHandler{
		service:      svc,
		mobileIssuer: mobileIssuer,
		socialAuth: service.NewSocialAuthService(
			svc,
			mobileIssuer,
			socialLinkTokens,
			socialLifecycle,
			service.NewKakaoIdentityVerifier(svc),
			appleVerifier,
		),
		appleVerifier:    appleVerifier,
		socialLifecycle:  socialLifecycle,
		memberSvc:        memberSvc,
		registerSvc:      registerSvc,
		profileSvc:       profileSvc,
		cache:            cacheStore,
		socialLinkTokens: socialLinkTokens,
		cfg:              cfg,
		logger:           logger,
	}
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
	h.logger.Debug().Msg("kakao: redirecting to authorize")
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
	h.logger.Debug().Msg("kakao: callback received")
	info, err := h.service.ExchangeKakaoToken(code)
	if err != nil {
		h.logger.Error().Err(err).Msg("kakao: token exchange failed")
		respondError(w, http.StatusBadRequest, "KAKAO_EXCHANGE_FAILED", "Kakao token exchange failed")
		return
	}
	h.logger.Debug().
		Bool("has_email", info.Email != "").
		Bool("has_profile_image", info.ProfileImageURL != "").
		Msg("kakao: token exchanged")
	h.handleSocialCallback(w, r, "KT", info)
}

func (h *AuthHandler) MobileLogin(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error().Err(err).Str("usrId", req.USRID).Msg("mobile login: password verification failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	if user == nil {
		respondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "아이디 또는 비밀번호가 올바르지 않습니다")
		return
	}
	if err := h.service.LoginWithBridge(user, w, r); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("mobile login: bridge session failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 처리 중 오류가 발생했습니다")
		return
	}
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	mobileSessionID := h.service.GenerateSessionID()
	if mobileSessionID == "" {
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 생성에 실패했습니다")
		return
	}
	mobileToken, err := h.service.GenerateMobileJWT(&authUser, mobileSessionID)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("mobile login: token issue failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 발급에 실패했습니다")
		return
	}
	refreshToken, refreshJTI, refreshExpiresAt, err := h.service.GenerateMobileRefreshJWT(&authUser, mobileSessionID)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("mobile login: refresh token issue failed")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 재발급에 실패했습니다")
		return
	}
	if err := h.service.RecordMobileRefreshToken(authUser.USRSeq, mobileSessionID, refreshJTI, refreshExpiresAt); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("mobile login: failed to persist refresh token")
		respondError(w, http.StatusInternalServerError, "LOGIN_FAILED", "로그인 토큰 재발급에 실패했습니다")
		return
	}
	now := time.Now()
	respondJSON(w, http.StatusOK, struct {
		USRSeq           int    `json:"usrSeq"`
		USRID            string `json:"usrId"`
		USRName          string `json:"usrName"`
		USRStatus        string `json:"usrStatus"`
		AccessToken      string `json:"accessToken"`
		RefreshToken     string `json:"refreshToken"`
		AccessIssuedAt   int64  `json:"accessIssuedAt"`
		AccessExpiresAt  int64  `json:"accessExpiresAt"`
		RefreshExpiresAt int64  `json:"refreshExpiresAt"`
		Sid              string `json:"sid"`
		Jti              string `json:"jti"`
	}{
		USRSeq:           authUser.USRSeq,
		USRID:            authUser.USRID,
		USRName:          authUser.USRName,
		USRStatus:        authUser.USRStatus,
		AccessToken:      mobileToken,
		RefreshToken:     refreshToken,
		AccessIssuedAt:   now.Unix(),
		AccessExpiresAt:  now.Add(h.service.MobileAccessTokenTTL()).Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		Sid:              mobileSessionID,
		Jti:              refreshJTI,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh token is required")
		return
	}

	user, sid, refreshJTI, err := h.service.ValidateMobileRefreshToken(refreshToken)
	if err != nil || user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token")
		return
	}
	if ok, err := h.service.ConsumeMobileRefreshToken(user.USRSeq, refreshJTI); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("refresh: failed to consume refresh token")
		respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to refresh token")
		return
	} else if !ok {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token")
		return
	}

	if sid == "" {
		sid = h.service.GenerateSessionID()
	}

	newAccessToken, err := h.service.GenerateMobileJWT(user, sid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to reissue access token")
		return
	}

	newRefreshToken, newRefreshJTI, newRefreshExpAt, err := h.service.GenerateMobileRefreshJWT(user, sid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to reissue refresh token")
		return
	}
	if err := h.service.RecordMobileRefreshToken(user.USRSeq, sid, newRefreshJTI, newRefreshExpAt); err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to reissue refresh token")
		return
	}

	now := time.Now()
	respondJSON(w, http.StatusOK, struct {
		AccessToken      string `json:"accessToken"`
		RefreshToken     string `json:"refreshToken"`
		AccessIssuedAt   int64  `json:"accessIssuedAt"`
		AccessExpiresAt  int64  `json:"accessExpiresAt"`
		RefreshExpiresAt int64  `json:"refreshExpiresAt"`
		Sid              string `json:"sid"`
		Jti              string `json:"jti"`
	}{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		AccessIssuedAt:   now.Unix(),
		AccessExpiresAt:  now.Add(h.service.MobileAccessTokenTTL()).Unix(),
		RefreshExpiresAt: newRefreshExpAt.Unix(),
		Sid:              sid,
		Jti:              newRefreshJTI,
	})
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
		if errors.Is(err, service.ErrLoginPending) || errors.Is(err, service.ErrLoginSuspended) || errors.Is(err, service.ErrLoginWithdrawn) {
			respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
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
	if req.UsrID == "" || req.Password == "" || req.Name == "" || req.Phone == "" || req.Email == "" || req.FN == "" || req.FmDept == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "필수 입력값이 누락되었습니다")
		return
	}
	if len(req.UsrID) < 4 || len(req.UsrID) > 20 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "아이디는 4~20자여야 합니다")
		return
	}
	if !fnDigitRegex.MatchString(req.FN) {
		respondError(w, http.StatusBadRequest, "INVALID_FN", "기수는 숫자로 입력해주세요")
		return
	}
	if !model.IsValidDepartment(req.FmDept) {
		respondError(w, http.StatusBadRequest, "INVALID_DEPARTMENT", "유효하지 않은 학과입니다")
		return
	}
	user, err := h.registerSvc.Register(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIDTaken):
			respondError(w, http.StatusConflict, "ID_TAKEN", "이미 사용 중인 아이디입니다")
		case errors.Is(err, service.ErrPhoneTaken):
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 등록된 전화번호입니다")
		case errors.Is(err, service.ErrEmailTaken):
			respondError(w, http.StatusConflict, "EMAIL_TAKEN", "이미 등록된 이메일입니다")
		case errors.Is(err, service.ErrTagContainsWhitespace):
			respondError(w, http.StatusBadRequest, "INVALID_TAG", "태그에 공백을 포함할 수 없습니다")
		default:
			h.logger.Error().Err(err).Msg("register: failed to create member")
			respondError(w, http.StatusInternalServerError, "REGISTER_FAILED", "회원가입 처리 중 오류가 발생했습니다")
		}
		return
	}
	authUser := model.AuthUser{USRSeq: user.USRSeq, USRID: user.USRID, USRName: user.USRName, USRStatus: user.USRStatus}
	respondJSON(w, http.StatusCreated, authUser)
}

func (h *AuthHandler) CheckID(w http.ResponseWriter, r *http.Request) {
	usrID := r.URL.Query().Get("usrId")
	if usrID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_PARAM", "usrId 파라미터가 필요합니다")
		return
	}
	available, err := h.registerSvc.IsIDAvailable(usrID)
	if err != nil {
		h.logger.Error().Err(err).Str("usrId", usrID).Msg("check-id: db error")
		respondError(w, http.StatusInternalServerError, "CHECK_FAILED", "아이디 중복 확인에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"available": available})
}

func (h *AuthHandler) CheckPhone(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if phone == "" {
		respondJSON(w, http.StatusOK, map[string]bool{"available": false})
		return
	}
	available, err := h.registerSvc.IsPhoneAvailable(phone)
	if err != nil {
		h.logger.Error().Err(err).Msg("check-phone: db error")
		respondError(w, http.StatusInternalServerError, "CHECK_FAILED", "전화번호 중복 확인에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"available": available})
}

func (h *AuthHandler) CheckEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		respondJSON(w, http.StatusOK, map[string]bool{"available": false})
		return
	}
	available, err := h.registerSvc.IsEmailAvailable(email)
	if err != nil {
		h.logger.Error().Err(err).Str("email", email).Msg("check-email: db error")
		respondError(w, http.StatusInternalServerError, "CHECK_FAILED", "이메일 중복 확인에 실패했습니다")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"available": available})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	legacySessionID := ""
	if cookie, err := r.Cookie("DDusrSession_id"); err == nil {
		legacySessionID = cookie.Value
	}
	if err := h.service.LogoutCurrent(w, user, legacySessionID); err != nil {
		h.logger.Warn().Err(err).Int("usrSeq", user.USRSeq).Msg("current session logout failed")
		respondError(w, http.StatusServiceUnavailable, "LOGOUT_FAILED", "서버 세션 로그아웃에 실패했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	if err := h.service.LogoutAll(w, user.USRSeq); err != nil {
		h.logger.Warn().Err(err).Int("usrSeq", user.USRSeq).Msg("all session logout failed")
		respondError(w, http.StatusServiceUnavailable, "LOGOUT_FAILED", "전체 세션 로그아웃에 실패했습니다.")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetAuthUser(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		return
	}
	current, err := h.service.GetLoginAllowedUser(user.USRSeq)
	if err != nil || current == nil {
		respondError(w, http.StatusForbidden, service.LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다.")
		return
	}
	respondJSON(w, http.StatusOK, current)
}
