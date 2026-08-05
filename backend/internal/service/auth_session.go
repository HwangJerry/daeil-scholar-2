// auth_session.go — Session and cookie management: login bridge, logout, and user lookup
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// GenerateSessionID returns a cryptographically random 32-character hex string.
func (s *AuthService) GenerateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// LoginWithBridge issues JWT + legacy PHP cookies and records the login event.
func (s *AuthService) LoginWithBridge(user *model.User, w http.ResponseWriter, r *http.Request) error {
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		return err
	}
	principal, err := s.GetCurrentUser(user.USRSeq)
	if err != nil {
		return err
	}
	if principal == nil {
		return ErrLoginSuspended
	}
	if err := (LoginEligibilityPolicy{}).EnsureStatusAllowed(principal.USRStatus); err != nil {
		return err
	}

	sessionID := s.GenerateSessionID()
	if sessionID == "" {
		return errors.New("failed to generate session id")
	}
	token, err := s.GenerateJWT(principal)
	if err != nil {
		return err
	}
	if err := s.repo.RecordBridgeLogin(principal.USRSeq, sessionID, r.RemoteAddr, r.UserAgent()); err != nil {
		return err
	}

	secure := s.cfg.Server.IsSecure()
	http.SetCookie(w, &http.Cookie{
		Name:     "alumni_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.cfg.JWT.MaxAge.Seconds()),
	})
	legacyCookies := map[string]string{
		"DDusrSession_id": sessionID,
		"DDusrSEQ":        strconv.Itoa(principal.USRSeq),
		"DDusrID":         principal.USRID,
		"DDusrNAME":       principal.USRName,
		"DDusrSTATUS":     principal.USRStatus,
	}
	for name, value := range legacyCookies {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   0,
		})
	}
	return nil
}

// RecordMobileRefreshToken saves a refresh token state to allow one-time use enforcement.
func (s *AuthService) RecordMobileRefreshToken(usrSeq int, sid string, jti string, expiresAt time.Time, expectedStatus string) error {
	return s.repo.InsertMobileRefreshToken(usrSeq, sid, jti, expiresAt, expectedStatus)
}

// ConsumeMobileRefreshToken revokes a refresh token JTI so it cannot be reused.
// Returns true when one token row was revoked.
func (s *AuthService) ConsumeMobileRefreshToken(usrSeq int, jti string) (bool, error) {
	return s.repo.RevokeMobileRefreshToken(usrSeq, jti)
}

// RevokeAllMobileRefreshTokens logs out all active refresh tokens for a user.
func (s *AuthService) RevokeAllMobileRefreshTokens(usrSeq int) error {
	if usrSeq <= 0 {
		return nil
	}
	if err := s.repo.RevokeMobileRefreshTokensByUser(usrSeq); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) LogoutCurrent(w http.ResponseWriter, user *model.AuthUser) error {
	if user == nil {
		clearAuthCookies(w, s.cfg.Server.IsSecure())
		return nil
	}
	if user.SessionID != "" {
		if err := s.repo.RevokeMobileSession(user.USRSeq, user.SessionID); err != nil {
			return err
		}
	}
	if user.LegacySessionID != "" {
		if err := s.repo.DeleteLegacySession(user.USRSeq, user.LegacySessionID); err != nil {
			return err
		}
	}
	clearAuthCookies(w, s.cfg.Server.IsSecure())
	return nil
}

func (s *AuthService) LogoutAll(w http.ResponseWriter, usrSeq int) error {
	if err := s.LogoutKakao(usrSeq); err != nil {
		s.logger.Warn().Msg("kakao logout failed, proceeding with app logout all")
	}
	if err := s.repo.DeleteLegacySessionsByUser(usrSeq); err != nil {
		return err
	}
	if err := s.RevokeAllMobileRefreshTokens(usrSeq); err != nil {
		return err
	}
	clearAuthCookies(w, s.cfg.Server.IsSecure())
	return nil
}

func clearAuthCookies(w http.ResponseWriter, secure bool) {
	expire := func(name string) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			MaxAge:   -1,
		})
	}
	expire("alumni_token")
	expire("DDusrSession_id")
	expire("DDusrSEQ")
	expire("DDusrID")
	expire("DDusrNAME")
	expire("DDusrSTATUS")
}

// GetCurrentUser looks up the full auth user record by sequence number.
func (s *AuthService) GetCurrentUser(usrSeq int) (*model.AuthUser, error) {
	return s.repo.GetAuthPrincipalBySeq(usrSeq)
}

// LookupLegacySession resolves a PHP DDusrSession_id cookie to an AuthUser.
func (s *AuthService) LookupLegacySession(sessionID string) (*model.AuthUser, error) {
	return s.repo.LookupLegacySession(sessionID)
}
