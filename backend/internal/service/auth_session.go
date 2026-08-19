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
	sessionID := s.GenerateSessionID()
	if sessionID == "" {
		return errors.New("failed to generate session id")
	}
	authUser := &model.AuthUser{
		USRSeq:    user.USRSeq,
		USRID:     user.USRID,
		USRName:   user.USRName,
		USRStatus: user.USRStatus,
	}
	token, err := s.GenerateJWT(authUser)
	if err != nil {
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
		"DDusrSEQ":        strconv.Itoa(user.USRSeq),
		"DDusrID":         user.USRID,
		"DDusrNAME":       user.USRName,
		"DDusrSTATUS":     user.USRStatus,
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
	if err := s.repo.InsertLoginLog(user.USRSeq, sessionID, r.RemoteAddr, r.UserAgent()); err != nil {
		return err
	}
	return s.repo.UpdateLastLogin(user.USRSeq)
}

// RecordMobileRefreshToken saves a refresh token state to allow one-time use enforcement.
func (s *AuthService) RecordMobileRefreshToken(usrSeq int, sid string, jti string, expiresAt time.Time) error {
	return s.repo.InsertMobileRefreshToken(usrSeq, sid, jti, expiresAt)
}

// ConsumeMobileRefreshToken revokes a refresh token JTI so it cannot be reused.
// Returns true when one token row was revoked.
func (s *AuthService) ConsumeMobileRefreshToken(usrSeq int, jti string) (bool, error) {
	return s.repo.RevokeMobileRefreshToken(usrSeq, jti)
}

// RevokeAllMobileRefreshTokens logs out all active refresh tokens for a user.
func (s *AuthService) RevokeAllMobileRefreshTokens(usrSeq int) {
	if usrSeq <= 0 {
		return
	}
	if err := s.repo.RevokeMobileRefreshTokensByUser(usrSeq); err != nil {
		s.logger.Warn().Err(err).Int("usrSeq", usrSeq).Msg("failed to revoke mobile refresh tokens on logout")
	}
}

// LogoutCurrent revokes only the mobile session represented by the request
// (this device's refresh tokens) plus the legacy PHP session, then clears cookies.
func (s *AuthService) LogoutCurrent(w http.ResponseWriter, user *model.AuthUser, legacySessionID string) error {
	var logoutErrors []error
	if err := s.LogoutKakao(user.USRSeq); err != nil {
		s.logger.Warn().Err(err).Int("usrSeq", user.USRSeq).Msg("kakao logout failed, proceeding with app logout")
	}
	if legacySessionID != "" {
		if err := s.repo.DeleteLegacySession(legacySessionID); err != nil {
			logoutErrors = append(logoutErrors, err)
		}
	}
	if user != nil && user.SessionID != "" {
		if err := s.repo.RevokeMobileRefreshTokensBySession(user.USRSeq, user.SessionID); err != nil {
			logoutErrors = append(logoutErrors, err)
		}
	}
	s.clearSessionCookies(w)
	return errors.Join(logoutErrors...)
}

// LogoutAll invalidates the Kakao access token (if cached), clears all legacy DB
// sessions, revokes every mobile refresh token for the user, then clears cookies.
func (s *AuthService) LogoutAll(w http.ResponseWriter, usrSeq int) error {
	var logoutErrors []error
	if err := s.LogoutKakao(usrSeq); err != nil {
		s.logger.Warn().Err(err).Int("usrSeq", usrSeq).Msg("kakao logout failed, proceeding with app logout")
	}
	if err := s.repo.DeleteLegacySessionsByUser(usrSeq); err != nil {
		logoutErrors = append(logoutErrors, err)
	}
	s.RevokeAllMobileRefreshTokens(usrSeq)
	s.clearSessionCookies(w)
	return errors.Join(logoutErrors...)
}

// Logout is retained for web callers compiled against the previous service API.
func (s *AuthService) Logout(w http.ResponseWriter, usrSeq int) {
	_ = s.LogoutAll(w, usrSeq)
}

func (s *AuthService) clearSessionCookies(w http.ResponseWriter) {
	secure := s.cfg.Server.IsSecure()
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
	user, err := s.repo.GetMemberBySeq(usrSeq)
	if err != nil || user == nil {
		return nil, err
	}
	return &model.AuthUser{
		USRSeq:    user.USRSeq,
		USRID:     user.USRID,
		USRName:   user.USRName,
		USRStatus: user.USRStatus,
	}, nil
}

func (s *AuthService) GetLoginAllowedUser(usrSeq int) (*model.AuthUser, error) {
	user, err := s.repo.GetMemberBySeq(usrSeq)
	if err != nil {
		return nil, err
	}
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	authUser := authUserFromMember(user)
	return &authUser, nil
}

// LookupLegacySession resolves a PHP DDusrSession_id cookie to an AuthUser.
func (s *AuthService) LookupLegacySession(sessionID string) (*model.AuthUser, error) {
	return s.repo.LookupLegacySession(sessionID)
}
