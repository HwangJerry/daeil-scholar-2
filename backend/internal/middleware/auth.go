package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

const jwtCookieName = "alumni_token"

func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := resolveAuthUser(authService, r, false)
			if err == nil && user != nil {
				ctx := SetAuthUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
		})
	}
}

func OptionalAuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := resolveAuthUser(authService, r, false)
			if err == nil && user != nil {
				ctx := SetAuthUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func resolveAuthUser(authService *service.AuthService, r *http.Request, strictHeader bool) (*model.AuthUser, error) {
	user, usedHeader, err := resolveAuthFromAuthorizationHeader(authService, r)
	if err == nil && user != nil {
		return user, nil
	}
	if strictHeader && usedHeader {
		return nil, errors.New("unauthorized")
	}

	if user, err := resolveAuthFromCookies(authService, r); err == nil && user != nil {
		return user, nil
	}

	if !usedHeader && err != nil {
		return nil, nil
	}

	return nil, errors.New("unauthorized")
}

func resolveAuthFromAuthorizationHeader(authService *service.AuthService, r *http.Request) (*model.AuthUser, bool, error) {
	headerToken, ok := extractBearerToken(r)
	if !ok {
		return nil, false, errors.New("missing bearer token")
	}

	mobileTokenType, isMobileToken, classifyErr := authService.ClassifyMobileToken(headerToken)
	if isMobileToken {
		if classifyErr != nil {
			return nil, true, errors.New("invalid mobile token")
		}

		if mobileTokenType != "access" {
			return nil, true, errors.New("unsupported mobile token type")
		}

		user, err := authService.ValidateMobileAccessToken(headerToken)
		if err != nil || user == nil {
			return nil, true, errors.New("invalid mobile token")
		}

		// Mobile-issued tokens carry a session id (sid); re-check on every
		// request that the session hasn't been logged out/revoked, so a
		// stolen or revoked access token stops working immediately instead
		// of staying valid until natural JWT expiry. Legacy web JWTs have no
		// SessionID and are unaffected by this check.
		if user.SessionID != "" {
			active, sessionErr := authService.IsMobileSessionActive(user.USRSeq, user.SessionID)
			if sessionErr != nil || !active {
				return nil, true, errors.New("mobile session revoked")
			}
		}

		return user, true, nil
	}

	if user, err := authService.ValidateJWT(headerToken); err == nil && user != nil {
		return user, true, nil
	}

	return nil, true, errors.New("invalid bearer token")
}

func resolveAuthFromCookies(authService *service.AuthService, r *http.Request) (*model.AuthUser, error) {
	if cookie, err := r.Cookie(jwtCookieName); err == nil && cookie.Value != "" {
		user, jwtErr := authService.ValidateJWT(cookie.Value)
		if jwtErr == nil && user != nil {
			return user, nil
		}
	}

	legacyCookie, err := r.Cookie("DDusrSession_id")
	if err == nil && legacyCookie.Value != "" {
		legacyUser, legacyErr := authService.LookupLegacySession(legacyCookie.Value)
		if legacyErr == nil && legacyUser != nil && (legacyUser.USRStatus == "CCC" || legacyUser.USRStatus == "ZZZ") {
			return legacyUser, nil
		}
	}

	return nil, errors.New("unauthorized")
}

func extractBearerToken(r *http.Request) (string, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", false
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}

	return strings.TrimSpace(parts[1]), true
}

func setJWTCookie(w http.ResponseWriter, token string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     jwtCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

func setLegacyCookie(w http.ResponseWriter, name string, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   0,
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	clearCookie(w, jwtCookieName)
	clearCookie(w, "DDusrSession_id")
	clearCookie(w, "DDusrSEQ")
	clearCookie(w, "DDusrID")
	clearCookie(w, "DDusrNAME")
	clearCookie(w, "DDusrSTATUS")
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})
}

func respondError(w http.ResponseWriter, status int, code string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIError{Code: code, Message: msg})
}
