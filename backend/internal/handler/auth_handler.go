package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
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
	cache            *cache.Cache
	socialLinkTokens *service.SocialLinkTokenStore
	phoneVerifier    *service.PhoneVerificationService
	consentSvc       *service.ConsentService
	cfg              *config.Config
	logger           zerolog.Logger
}

// AttachPhoneVerification makes signup require a verified phone number. Registration
// endpoints reject requests without a usable grant token once this is set.
func (h *AuthHandler) AttachPhoneVerification(verifier *service.PhoneVerificationService) {
	h.phoneVerifier = verifier
}

// AttachPrivacyConsent makes signup evaluate and record the data-collection consent.
// Without it, consent fields are ignored (pre-rollout behaviour).
func (h *AuthHandler) AttachPrivacyConsent(consentSvc *service.ConsentService) {
	h.consentSvc = consentSvc
}

// requirePrivacyConsent applies the consent policy before an account is created.
// Returns false after writing the error response.
func (h *AuthHandler) requirePrivacyConsent(w http.ResponseWriter, consent *model.PrivacyConsent) bool {
	if h.consentSvc == nil {
		return true
	}
	err := h.consentSvc.Evaluate(consent)
	switch {
	case err == nil:
		return true
	case errors.Is(err, service.ErrConsentRequired):
		respondError(w, http.StatusBadRequest, "CONSENT_REQUIRED", "개인정보 수집·이용에 동의해야 가입할 수 있습니다")
	case errors.Is(err, service.ErrConsentVersionOutdated):
		respondError(w, http.StatusBadRequest, "CONSENT_VERSION_OUTDATED", "개인정보 안내 내용이 변경되었습니다. 앱을 최신 버전으로 업데이트한 뒤 다시 시도해주세요")
	default:
		h.logger.Error().Err(err).Msg("register: consent evaluation failed")
		respondError(w, http.StatusInternalServerError, "CONSENT_CHECK_FAILED", "동의 확인 중 오류가 발생했습니다")
	}
	return false
}

// recordPrivacyConsent stores the consent after the account exists. Like the phone
// grant, a failure here must not undo the signup, so it is logged.
func (h *AuthHandler) recordPrivacyConsent(accountID int, consent *model.PrivacyConsent) {
	if h.consentSvc == nil {
		return
	}
	if err := h.consentSvc.Record(accountID, consent); err != nil {
		h.logger.Error().Err(err).Int("usrSeq", accountID).Msg("register: failed to record privacy consent")
	}
}

// requirePhoneVerification confirms a grant exists for the phone number without
// spending it. Returns false after writing the error response.
func (h *AuthHandler) requirePhoneVerification(w http.ResponseWriter, token, phone string) bool {
	if h.phoneVerifier == nil {
		return true
	}
	err := h.phoneVerifier.AssertPhoneVerified(token, phone)
	switch {
	case err == nil:
		return true
	case errors.Is(err, service.ErrPhoneNotVerified):
		respondError(w, http.StatusBadRequest, "PHONE_NOT_VERIFIED", "휴대폰 인증을 먼저 완료해주세요")
	case errors.Is(err, service.ErrInvalidPhone):
		respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
	default:
		h.logger.Error().Err(err).Msg("register: phone verification check failed")
		respondError(w, http.StatusInternalServerError, "PHONE_VERIFICATION_FAILED", "휴대폰 인증 확인에 실패했습니다")
	}
	return false
}

// spendPhoneVerification marks the grant used after the account exists. A failure here
// leaves an already-created account intact, so it is logged rather than surfaced.
func (h *AuthHandler) spendPhoneVerification(token, phone string) {
	if h.phoneVerifier == nil {
		return
	}
	if err := h.phoneVerifier.ConsumeGrantForPhone(token, phone); err != nil {
		h.logger.Error().Err(err).Msg("register: failed to consume phone verification grant")
	}
}

func NewAuthHandler(
	svc *service.AuthService,
	memberSvc *service.MemberService,
	registerSvc *service.RegistrationService,
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
	if !h.requirePrivacyConsent(w, req.PrivacyConsent) {
		return
	}
	if !h.requirePhoneVerification(w, req.PhoneVerificationToken, req.Phone) {
		return
	}
	user, err := h.registerSvc.Register(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIDTaken):
			respondError(w, http.StatusConflict, "ID_TAKEN", "이미 사용 중인 아이디입니다")
		case errors.Is(err, service.ErrPhonePendingDeletion):
			respondError(w, http.StatusConflict, "PHONE_PENDING_DELETION", "탈퇴 처리 중인 전화번호입니다. 탈퇴 처리가 끝난 뒤 다시 가입해주세요")
		case errors.Is(err, service.ErrPhoneTaken):
			respondError(w, http.StatusConflict, "PHONE_TAKEN", "이미 등록된 전화번호입니다")
		case errors.Is(err, service.ErrInvalidPhone):
			respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
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
	h.spendPhoneVerification(req.PhoneVerificationToken, req.Phone)
	h.recordPrivacyConsent(user.USRSeq, req.PrivacyConsent)
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
		if errors.Is(err, service.ErrInvalidPhone) {
			respondError(w, http.StatusBadRequest, "INVALID_PHONE", "유효한 전화번호를 입력해주세요")
			return
		}
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
	w.WriteHeader(http.StatusNoContent)
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
	w.WriteHeader(http.StatusNoContent)
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
