// auth_session.go — Session and cookie management: login bridge, logout, and user lookup
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

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
			MaxAge:   0,
		})
	}
	if err := s.repo.InsertLoginLog(user.USRSeq, sessionID, r.RemoteAddr, r.UserAgent()); err != nil {
		return err
	}
	return s.repo.UpdateLastLogin(user.USRSeq)
}

// Logout invalidates the Kakao access token (if cached) then clears all session cookies.
func (s *AuthService) Logout(w http.ResponseWriter, usrSeq int) {
	if err := s.LogoutKakao(usrSeq); err != nil {
		s.logger.Warn().Err(err).Int("usrSeq", usrSeq).Msg("kakao logout failed, proceeding with app logout")
	}
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

// LookupLegacySession resolves a PHP DDusrSession_id cookie to an AuthUser.
func (s *AuthService) LookupLegacySession(sessionID string) (*model.AuthUser, error) {
	return s.repo.LookupLegacySession(sessionID)
}
