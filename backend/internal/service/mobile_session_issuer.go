package service

import (
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type MobileSessionIssuer struct {
	auth   *AuthService
	policy LoginEligibilityPolicy
	now    func() time.Time
}

func NewMobileSessionIssuer(auth *AuthService) *MobileSessionIssuer {
	return &MobileSessionIssuer{
		auth: auth,
		now:  time.Now,
	}
}

func (i *MobileSessionIssuer) Issue(user *model.User) (*model.MobileSession, error) {
	if err := i.policy.EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	now := i.now()
	sid := i.auth.GenerateSessionID()
	if sid == "" {
		return nil, errors.New("failed to generate session id")
	}
	authUser := authUserFromMember(user)
	accessToken, accessExpiresAt, err := i.auth.generateMobileAccessToken(&authUser, sid, now)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshJTI, refreshExpiresAt, err := i.auth.generateMobileRefreshToken(&authUser, sid, now)
	if err != nil {
		return nil, err
	}
	if err := i.auth.repo.InsertMobileRefreshToken(user.USRSeq, sid, refreshJTI, refreshExpiresAt); err != nil {
		return nil, err
	}
	return buildMobileSession(authUser, accessToken, refreshToken, sid, refreshJTI, now, accessExpiresAt, refreshExpiresAt), nil
}

func (i *MobileSessionIssuer) Rotate(refreshToken string) (*model.MobileSession, error) {
	claimsUser, oldJTI, err := i.auth.ValidateMobileRefreshToken(refreshToken)
	if err != nil {
		return nil, repository.ErrRefreshTokenInvalid
	}
	currentUser, err := i.auth.repo.GetMemberBySeq(claimsUser.USRSeq)
	if err != nil {
		return nil, err
	}
	if err := i.policy.EnsureLoginAllowed(currentUser); err != nil {
		_ = i.auth.repo.RevokeMobileSession(claimsUser.USRSeq, claimsUser.SessionID)
		return nil, err
	}

	now := i.now()
	authUser := authUserFromMember(currentUser)
	accessToken, accessExpiresAt, err := i.auth.generateMobileAccessToken(&authUser, claimsUser.SessionID, now)
	if err != nil {
		return nil, err
	}
	newRefreshToken, newJTI, refreshExpiresAt, err := i.auth.generateMobileRefreshToken(&authUser, claimsUser.SessionID, now)
	if err != nil {
		return nil, err
	}
	if err := i.auth.repo.RotateMobileRefreshToken(
		currentUser.USRSeq,
		claimsUser.SessionID,
		oldJTI,
		newJTI,
		refreshExpiresAt,
	); err != nil {
		return nil, err
	}
	return buildMobileSession(authUser, accessToken, newRefreshToken, claimsUser.SessionID, newJTI, now, accessExpiresAt, refreshExpiresAt), nil
}

func (i *MobileSessionIssuer) RevokeCurrent(user *model.AuthUser) error {
	if user == nil || user.USRSeq <= 0 || user.SessionID == "" {
		return nil
	}
	return i.auth.repo.RevokeMobileSession(user.USRSeq, user.SessionID)
}

func (i *MobileSessionIssuer) RevokeAll(usrSeq int) error {
	if usrSeq <= 0 {
		return nil
	}
	return i.auth.repo.RevokeAllMobileSessions(usrSeq)
}

func authUserFromMember(user *model.User) model.AuthUser {
	return model.AuthUser{
		USRSeq:    user.USRSeq,
		USRID:     user.USRID,
		USRName:   user.USRName,
		USRStatus: user.USRStatus,
	}
}

func buildMobileSession(
	user model.AuthUser,
	accessToken string,
	refreshToken string,
	sid string,
	jti string,
	issuedAt time.Time,
	accessExpiresAt time.Time,
	refreshExpiresAt time.Time,
) *model.MobileSession {
	return &model.MobileSession{
		User:             user,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessIssuedAt:   issuedAt.Unix(),
		AccessExpiresAt:  accessExpiresAt.Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		SID:              sid,
		JTI:              jti,
	}
}
