package service

import (
	"context"
	"errors"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

type VerifiedSocialAccount struct {
	Identity        model.VerifiedSocialIdentity
	Profile         model.SocialProviderProfile
	ProviderToken   string
	RevocationToken string
}

type SocialIdentityVerifier interface {
	Provider() model.SocialProvider
	Verify(ctx context.Context, authorization model.SocialAuthorization) (VerifiedSocialAccount, error)
}

type KakaoIdentityVerifier struct {
	auth *AuthService
}

func NewKakaoIdentityVerifier(auth *AuthService) *KakaoIdentityVerifier {
	return &KakaoIdentityVerifier{auth: auth}
}

func (*KakaoIdentityVerifier) Provider() model.SocialProvider {
	return model.SocialProviderKakao
}

func (v *KakaoIdentityVerifier) Verify(_ context.Context, authorization model.SocialAuthorization) (VerifiedSocialAccount, error) {
	kakaoAuthorization, ok := authorization.(model.KakaoAuthorization)
	if !ok || strings.TrimSpace(kakaoAuthorization.AccessToken) == "" {
		return VerifiedSocialAccount{}, errors.New("missing kakao access token")
	}
	info, err := v.auth.GetKakaoProfileByAccessToken(kakaoAuthorization.AccessToken)
	if err != nil {
		return VerifiedSocialAccount{}, err
	}
	return VerifiedSocialAccount{
		Identity: model.VerifiedSocialIdentity{
			Provider:      model.SocialProviderKakao,
			Subject:       info.KakaoID,
			Email:         info.Email,
			EmailVerified: info.Email != "",
		},
		Profile: model.SocialProviderProfile{
			DisplayName:     info.Nickname,
			Email:           info.Email,
			ProfileImageURL: info.ProfileImageURL,
		},
		ProviderToken: info.AccessToken,
	}, nil
}

type SocialAuthService struct {
	auth       *AuthService
	issuer     *MobileSessionIssuer
	linkTokens *SocialLinkTokenStore
	verifiers  map[model.SocialProvider]SocialIdentityVerifier
	policy     LoginEligibilityPolicy
	lifecycle  *SocialAccountLifecycleService
}

func NewSocialAuthService(
	auth *AuthService,
	issuer *MobileSessionIssuer,
	linkTokens *SocialLinkTokenStore,
	lifecycle *SocialAccountLifecycleService,
	verifiers ...SocialIdentityVerifier,
) *SocialAuthService {
	byProvider := make(map[model.SocialProvider]SocialIdentityVerifier, len(verifiers))
	for _, verifier := range verifiers {
		byProvider[verifier.Provider()] = verifier
	}
	return &SocialAuthService{
		auth:       auth,
		issuer:     issuer,
		linkTokens: linkTokens,
		lifecycle:  lifecycle,
		verifiers:  byProvider,
	}
}

func (s *SocialAuthService) Authenticate(
	ctx context.Context,
	authorization model.SocialAuthorization,
) (model.SocialAuthResult, error) {
	if authorization == nil {
		return rejectedSocialResult("PROVIDER_NOT_SUPPORTED", "지원하지 않는 로그인 제공자입니다."), nil
	}
	provider := authorization.Provider()
	verifier, ok := s.verifiers[provider]
	if !ok {
		return rejectedSocialResult("PROVIDER_NOT_SUPPORTED", "지원하지 않는 로그인 제공자입니다."), nil
	}
	account, err := verifier.Verify(ctx, authorization)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	user, err := s.auth.FindMemberBySocialID(string(account.Identity.Provider), account.Identity.Subject)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	if user != nil {
		return s.resultForLinkedUser(user, account)
	}

	linkToken := s.auth.GenerateSessionID()
	if linkToken == "" {
		return model.SocialAuthResult{}, errors.New("failed to create link token")
	}
	expiresAt, err := s.linkTokens.Put(linkToken, model.SocialLinkData{
		Provider:        string(account.Identity.Provider),
		SocialID:        account.Identity.Subject,
		Email:           account.Identity.Email,
		Nickname:        account.Profile.DisplayName,
		ProfileImageURL: account.Profile.ProfileImageURL,
		AccessToken:     account.ProviderToken,
		RevocationToken: account.RevocationToken,
	}, SocialLinkTokenTTL)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	return model.SocialAuthResult{
		Status: model.SocialAuthLinkRequired,
		LinkRequired: &model.SocialLinkContext{
			LinkToken: linkToken,
			Provider:  account.Identity.Provider,
			Profile:   account.Profile,
			ExpiresAt: expiresAt.Unix(),
		},
	}, nil
}

func (s *SocialAuthService) resultForLinkedUser(
	user *model.User,
	account VerifiedSocialAccount,
) (model.SocialAuthResult, error) {
	if err := s.policy.EnsureLoginAllowed(user); err != nil {
		if errors.Is(err, ErrLoginPending) {
			pending := authUserFromMember(user)
			return model.SocialAuthResult{
				Status:  model.SocialAuthPending,
				Pending: &pending,
			}, nil
		}
		return rejectedSocialResult(LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다."), nil
	}
	credential := account.ProviderToken
	if account.Identity.Provider == model.SocialProviderApple {
		credential = account.RevocationToken
	}
	if credential != "" && s.lifecycle != nil {
		if err := s.lifecycle.StoreCredential(user.USRSeq, account.Identity.Provider, credential); err != nil {
			return model.SocialAuthResult{}, err
		}
	}
	session, err := s.issuer.Issue(user)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	if account.Identity.Provider == model.SocialProviderKakao && account.ProviderToken != "" {
		s.auth.CacheKakaoToken(user.USRSeq, account.ProviderToken)
	}
	return model.SocialAuthResult{
		Status:  model.SocialAuthAuthenticated,
		Session: session,
	}, nil
}

func (s *SocialAuthService) CompleteMobileLink(user *model.User) (model.SocialAuthResult, error) {
	if err := s.policy.EnsureLoginAllowed(user); err != nil {
		if errors.Is(err, ErrLoginPending) {
			pending := authUserFromMember(user)
			return model.SocialAuthResult{
				Status:  model.SocialAuthPending,
				Pending: &pending,
			}, nil
		}
		return rejectedSocialResult(LoginErrorCode(err), "이 계정은 현재 로그인할 수 없습니다."), nil
	}
	session, err := s.issuer.Issue(user)
	if err != nil {
		return model.SocialAuthResult{}, err
	}
	return model.SocialAuthResult{
		Status:  model.SocialAuthAuthenticated,
		Session: session,
	}, nil
}

func rejectedSocialResult(code string, message string) model.SocialAuthResult {
	return model.SocialAuthResult{
		Status: model.SocialAuthRejected,
		Rejected: &model.SocialAuthRejection{
			Code:    code,
			Message: message,
		},
	}
}
