package service

import (
	"context"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var (
	ErrInvalidSocialProvider      = errors.New("invalid social provider")
	ErrLastLoginMethod            = errors.New("cannot disconnect the last login method")
	ErrSocialIdentityVerification = errors.New("social identity verification failed")
	ErrSocialCredentialStorage    = errors.New("social credential storage unavailable")
)

func (s *SocialAuthService) LinkIdentity(
	ctx context.Context,
	usrSeq int,
	provider string,
	authorization model.SocialAuthorization,
) (model.AccountConnections, error) {
	socialProvider, err := socialProviderFromPath(provider)
	if err != nil {
		return model.AccountConnections{}, err
	}
	if authorization == nil || authorization.Provider() != socialProvider {
		return model.AccountConnections{}, ErrInvalidSocialProvider
	}

	verifier, ok := s.verifiers[socialProvider]
	if !ok {
		return model.AccountConnections{}, ErrInvalidSocialProvider
	}
	account, err := verifier.Verify(ctx, authorization)
	if err != nil {
		return model.AccountConnections{}, errors.Join(ErrSocialIdentityVerification, err)
	}
	if account.Identity.Provider != socialProvider || account.Identity.Subject == "" {
		return model.AccountConnections{}, ErrSocialIdentityVerification
	}

	existing, err := s.auth.FindMemberBySocialID(string(socialProvider), account.Identity.Subject)
	if err != nil {
		return model.AccountConnections{}, err
	}
	if existing != nil {
		if existing.USRSeq != usrSeq {
			return model.AccountConnections{}, ErrSocialAccountAlreadyLinked
		}
		return s.auth.GetAccountConnections(usrSeq)
	}

	encryptedCredential, err := s.encryptedCredentialForLink(account)
	if err != nil {
		return model.AccountConnections{}, err
	}
	err = s.auth.repo.LinkSocialIdentity(repository.SocialAccountFields{
		USRSeq:              usrSeq,
		Provider:            string(socialProvider),
		SocialID:            account.Identity.Subject,
		SocialEmail:         account.Identity.Email,
		Email:               account.Identity.Email,
		EncryptedCredential: encryptedCredential,
	})
	if errors.Is(err, repository.ErrSocialIdentityAlreadyLinked) {
		// A concurrent request may have inserted the same identity after the
		// lookup above. Preserve idempotency when that winner linked it here.
		existing, lookupErr := s.auth.FindMemberBySocialID(string(socialProvider), account.Identity.Subject)
		if lookupErr != nil {
			return model.AccountConnections{}, lookupErr
		}
		if existing != nil && existing.USRSeq == usrSeq {
			return s.auth.GetAccountConnections(usrSeq)
		}
		return model.AccountConnections{}, ErrSocialAccountAlreadyLinked
	}
	if err != nil {
		return model.AccountConnections{}, err
	}

	if socialProvider == model.SocialProviderKakao && account.ProviderToken != "" {
		s.auth.CacheKakaoToken(usrSeq, account.ProviderToken)
	}
	return s.auth.GetAccountConnections(usrSeq)
}

func (s *SocialAuthService) encryptedCredentialForLink(account VerifiedSocialAccount) (string, error) {
	if s.lifecycle == nil {
		return "", nil
	}
	credential := account.ProviderToken
	if account.Identity.Provider == model.SocialProviderApple {
		credential = account.RevocationToken
	}
	encrypted, err := s.lifecycle.EncryptCredential(credential)
	if err != nil {
		return "", errors.Join(ErrSocialCredentialStorage, err)
	}
	return encrypted, nil
}

func (s *AuthService) GetAccountConnections(usrSeq int) (model.AccountConnections, error) {
	return s.repo.GetAccountConnections(usrSeq)
}

func (s *AuthService) Disconnect(usrSeq int, provider string) (model.SocialDisconnectResult, error) {
	socialProvider, err := socialProviderFromPath(provider)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}

	connections, err := s.GetAccountConnections(usrSeq)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}
	if !hasSocialProvider(connections.Providers, socialProvider) {
		return model.SocialDisconnectResult{
			Status:      model.SocialDisconnectStatusNotConnected,
			Connections: connections,
		}, nil
	}

	if err := s.repo.DeleteSocialConnection(usrSeq, string(socialProvider)); err != nil {
		if errors.Is(err, repository.ErrLastLoginMethod) {
			return model.SocialDisconnectResult{}, ErrLastLoginMethod
		}
		return model.SocialDisconnectResult{}, err
	}

	updatedConnections, err := s.GetAccountConnections(usrSeq)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}
	return model.SocialDisconnectResult{
		Status:      model.SocialDisconnectStatusDisconnected,
		Connections: updatedConnections,
	}, nil
}

func socialProviderFromPath(provider string) (model.SocialProvider, error) {
	switch provider {
	case "kakao":
		return model.SocialProviderKakao, nil
	case "apple":
		return model.SocialProviderApple, nil
	default:
		return "", ErrInvalidSocialProvider
	}
}

func hasSocialProvider(providers []string, target model.SocialProvider) bool {
	for _, provider := range providers {
		if provider == string(target) {
			return true
		}
	}
	return false
}
