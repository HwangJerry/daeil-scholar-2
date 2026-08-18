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
			user, err := resolveAuthUser(authService, r)
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
			user, err := resolveAuthUser(authService, r)
			if err == nil && user != nil {
				ctx := SetAuthUser(r.Context(), user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func resolveAuthUser(authService *service.AuthService, r *http.Request) (*model.AuthUser, error) {
	if bearerToken, ok := extractBearerToken(r.Header.Get("Authorization")); ok {
		user, err := authService.ValidateMobileAccessToken(bearerToken)
		if err != nil {
			return nil, errors.New("unauthorized")
		}
		current, err := authService.GetLoginAllowedUser(user.USRSeq)
		if err != nil || current == nil {
			return nil, errors.New("unauthorized")
		}
		active, err := authService.IsMobileSessionActive(user.USRSeq, user.SessionID)
		if err != nil || !active {
			return nil, errors.New("unauthorized")
		}
		current.SessionID = user.SessionID
		return current, nil
	}

	cookie, err := r.Cookie(jwtCookieName)
	if err == nil && cookie.Value != "" {
		user, jwtErr := authService.ValidateJWT(cookie.Value)
		if jwtErr == nil && user != nil {
			return authService.GetLoginAllowedUser(user.USRSeq)
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

func extractBearerToken(header string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
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
