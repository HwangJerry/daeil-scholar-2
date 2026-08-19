package service

import (
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// MobileSessionIssuer issues mobile JWT sessions for a verified member using the
// same access/refresh token model as the legacy ID/password and Kakao mobile
// login handlers (GenerateMobileJWT + GenerateMobileRefreshJWT + RecordMobileRefreshToken).
// Token refresh/rotation is provider-agnostic and handled entirely by
// AuthHandler.Refresh, so this issuer only covers initial session issuance.
type MobileSessionIssuer struct {
	auth   *AuthService
	policy LoginEligibilityPolicy
}

func NewMobileSessionIssuer(auth *AuthService) *MobileSessionIssuer {
	return &MobileSessionIssuer{auth: auth}
}

func (i *MobileSessionIssuer) Issue(user *model.User) (*model.MobileSession, error) {
	if err := i.policy.EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	sid := i.auth.GenerateSessionID()
	if sid == "" {
		return nil, errors.New("failed to generate session id")
	}
	authUser := authUserFromMember(user)

	accessToken, err := i.auth.GenerateMobileJWT(&authUser, sid)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshJTI, refreshExpiresAt, err := i.auth.GenerateMobileRefreshJWT(&authUser, sid)
	if err != nil {
		return nil, err
	}
	if err := i.auth.RecordMobileRefreshToken(user.USRSeq, sid, refreshJTI, refreshExpiresAt); err != nil {
		return nil, err
	}

	now := time.Now()
	return &model.MobileSession{
		User:             authUser,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessIssuedAt:   now.Unix(),
		AccessExpiresAt:  now.Add(i.auth.cfg.JWT.MaxAge).Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		SID:              sid,
		JTI:              refreshJTI,
	}, nil
}

func authUserFromMember(user *model.User) model.AuthUser {
	return model.AuthUser{
		USRSeq:    user.USRSeq,
		USRID:     user.USRID,
		USRName:   user.USRName,
		USRStatus: user.USRStatus,
	}
}
