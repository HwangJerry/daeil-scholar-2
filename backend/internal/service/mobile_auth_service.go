// mobile_auth_service.go — Canonical mobile social-auth orchestration.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/social-auth/kakao"
)

const socialLinkContinuationTTL = 5 * time.Minute

var ErrKakaoVerificationFailed = errors.New("kakao verification failed")

func (s *AuthService) AuthenticateKakaoMobile(ctx context.Context, accessToken string) (model.SocialAuthResult, error) {
	authResult, err := s.kakaoClient.AuthenticateByAccessToken(ctx, accessToken)
	if err != nil {
		return model.SocialAuthResult{}, ErrKakaoVerificationFailed
	}
	return s.authenticateKakaoResult(authResult)
}

func (s *AuthService) authenticateKakaoResult(authResult kakao.AuthResult) (model.SocialAuthResult, error) {
	user, err := s.repo.FindMemberByKakaoID(authResult.Profile.KakaoID)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	if user == nil {
		linkRequired, err := s.BeginSocialLinkContinuation(
			model.SocialProviderKakao,
			authResult.Profile.KakaoID,
			authResult.Profile.Email,
			authResult.Profile.Nickname,
			authResult.Profile.ProfileImageURL,
		)
		if err != nil {
			return model.SocialAuthResult{}, err
		}
		return model.SocialAuthResult{
			Status:       model.SocialAuthLinkRequired,
			LinkRequired: linkRequired,
		}, nil
	}
	session, err := s.IssueMobileSession(user)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	return model.SocialAuthResult{
		Status:  model.SocialAuthAuthenticated,
		Session: session,
	}, nil
}

func (s *AuthService) BeginSocialLinkContinuation(
	provider model.SocialProvider,
	providerID string,
	email string,
	displayName string,
	avatarURL string,
) (*model.SocialLinkContext, error) {
	linkToken := s.GenerateSessionID()
	if linkToken == "" {
		return nil, errors.New("failed to generate social link token")
	}
	expiresAt := time.Now().Add(socialLinkContinuationTTL)
	if err := s.repo.InsertSocialLinkContinuation(
		hashSocialLinkToken(linkToken),
		string(provider),
		providerID,
		email,
		expiresAt,
	); err != nil {
		return nil, err
	}
	return &model.SocialLinkContext{
		LinkToken: linkToken,
		Provider:  provider,
		Profile: model.SocialProviderProfile{
			DisplayName: displayName,
			Email:       email,
			AvatarURL:   avatarURL,
		},
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *AuthService) issueCanonicalMobileSession(user *model.User) (*model.MobileSession, error) {
	principal, err := s.repo.GetAuthPrincipalBySeq(user.USRSeq)
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, errors.New("canonical auth principal not found")
	}
	if err := (LoginEligibilityPolicy{}).EnsureStatusAllowed(principal.USRStatus); err != nil {
		return nil, err
	}

	issuedAt := time.Now()
	sid := s.GenerateSessionID()
	if sid == "" {
		return nil, errors.New("failed to generate session id")
	}
	accessToken, err := s.GenerateMobileJWT(principal, sid)
	if err != nil {
		return nil, err
	}
	refreshToken, jti, refreshExpiresAt, err := s.GenerateMobileRefreshJWT(principal, sid)
	if err != nil {
		return nil, err
	}
	if err := s.RecordMobileRefreshToken(user.USRSeq, sid, jti, refreshExpiresAt, principal.USRStatus); err != nil {
		return nil, err
	}

	responseUser := *principal
	responseUser.USRStatus = ""
	return &model.MobileSession{
		User:             responseUser,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessIssuedAt:   issuedAt.Unix(),
		AccessExpiresAt:  issuedAt.Add(s.cfg.JWT.MaxAge).Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		SID:              sid,
		JTI:              jti,
	}, nil
}

func (s *AuthService) IssueMobileSession(user *model.User) (*model.MobileSession, error) {
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	return s.issueCanonicalMobileSession(user)
}

func (s *AuthService) RotateMobileSession(refreshToken string) (*model.MobileSession, error) {
	claimsUser, sid, oldJTI, err := s.ValidateMobileRefreshToken(refreshToken)
	if err != nil || claimsUser == nil || sid == "" {
		return nil, repository.ErrRefreshTokenInvalid
	}
	currentUser, err := s.repo.GetMemberBySeq(claimsUser.USRSeq)
	if err != nil {
		return nil, err
	}
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(currentUser); err != nil {
		return nil, s.revokeIneligibleMobileSession(claimsUser.USRSeq, sid, err)
	}
	principal, err := s.repo.GetAuthPrincipalBySeq(currentUser.USRSeq)
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, s.revokeIneligibleMobileSession(claimsUser.USRSeq, sid, repository.ErrRefreshTokenInvalid)
	}
	if err := (LoginEligibilityPolicy{}).EnsureStatusAllowed(principal.USRStatus); err != nil {
		return nil, s.revokeIneligibleMobileSession(claimsUser.USRSeq, sid, err)
	}

	issuedAt := time.Now()
	accessToken, err := s.GenerateMobileJWT(principal, sid)
	if err != nil {
		return nil, err
	}
	newRefreshToken, newJTI, refreshExpiresAt, err := s.GenerateMobileRefreshJWT(principal, sid)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RotateMobileRefreshToken(currentUser.USRSeq, sid, oldJTI, newJTI, refreshExpiresAt, principal.USRStatus); err != nil {
		return nil, err
	}
	responseUser := *principal
	responseUser.USRStatus = ""
	return &model.MobileSession{
		User:             responseUser,
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		AccessIssuedAt:   issuedAt.Unix(),
		AccessExpiresAt:  issuedAt.Add(s.cfg.JWT.MaxAge).Unix(),
		RefreshExpiresAt: refreshExpiresAt.Unix(),
		SID:              sid,
		JTI:              newJTI,
	}, nil
}

func (s *AuthService) revokeIneligibleMobileSession(usrSeq int, sid string, reason error) error {
	if err := s.repo.RevokeMobileSession(usrSeq, sid); err != nil {
		return err
	}
	return reason
}

func (s *AuthService) IsMobileSessionActive(usrSeq int, sid string) (bool, error) {
	if usrSeq <= 0 || strings.TrimSpace(sid) == "" {
		return false, nil
	}
	return s.repo.IsMobileSessionActive(usrSeq, sid)
}

func (s *AuthService) CompleteCanonicalSocialLinkIdentity(linkToken string, email string, password string) (*model.User, error) {
	linkToken = strings.TrimSpace(linkToken)
	email = strings.TrimSpace(email)
	if linkToken == "" || email == "" || password == "" {
		return nil, repository.ErrSocialLinkTokenInvalid
	}
	user, err := s.repo.CompleteSocialLinkContinuation(
		hashSocialLinkToken(linkToken),
		email,
		MysqlNativePassword(password),
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) CompleteCanonicalSocialLink(linkToken string, email string, password string) (model.SocialAuthResult, error) {
	user, err := s.CompleteCanonicalSocialLinkIdentity(linkToken, email, password)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	session, err := s.IssueMobileSession(user)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	return model.SocialAuthResult{
		Status:  model.SocialAuthAuthenticated,
		Session: session,
	}, nil
}

func hashSocialLinkToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
