package service

import (
	"context"
	"errors"
	"sync"

	"github.com/dflh-saf/backend/internal/model"
)

const (
	revocationActionDisconnect = "DISCONNECT"
	revocationActionDelete     = "ACCOUNT_DELETE"
)

var ErrLastLoginMethod = errors.New("cannot disconnect the last login method")

type SocialAccountLifecycleService struct {
	auth         *AuthService
	apple        *AppleIdentityVerifier
	vault        *SocialCredentialVault
	vaultErr     error
	disconnectMu sync.Mutex
}

type AccountDeletionResult struct {
	RevocationPending bool
}

func NewSocialAccountLifecycleService(auth *AuthService, apple *AppleIdentityVerifier) *SocialAccountLifecycleService {
	vault, err := NewSocialCredentialVault(auth.cfg.Apple.CredentialEncryptionKey)
	return &SocialAccountLifecycleService{
		auth:     auth,
		apple:    apple,
		vault:    vault,
		vaultErr: err,
	}
}

func (s *SocialAccountLifecycleService) EnsureCredentialStorageAvailable(credential string) error {
	if credential == "" {
		return nil
	}
	if s.vaultErr != nil {
		return s.vaultErr
	}
	if !s.vault.Ready() {
		return ErrCredentialVaultNotConfigured
	}
	return nil
}

func (s *SocialAccountLifecycleService) StoreCredential(usrSeq int, provider model.SocialProvider, credential string) error {
	if credential == "" {
		return nil
	}
	if err := s.EnsureCredentialStorageAvailable(credential); err != nil {
		return err
	}
	encrypted, err := s.vault.Encrypt(credential)
	if err != nil {
		return err
	}
	return s.auth.repo.UpsertSocialCredential(usrSeq, string(provider), encrypted)
}

func (s *SocialAccountLifecycleService) EncryptCredential(credential string) (string, error) {
	if credential == "" {
		return "", nil
	}
	if err := s.EnsureCredentialStorageAvailable(credential); err != nil {
		return "", err
	}
	return s.vault.Encrypt(credential)
}

func (s *SocialAccountLifecycleService) Connections(usrSeq int) (model.AccountConnections, error) {
	return s.auth.repo.GetAccountConnections(usrSeq)
}

func (s *SocialAccountLifecycleService) Disconnect(
	ctx context.Context,
	usrSeq int,
	provider model.SocialProvider,
) (model.SocialDisconnectResult, error) {
	s.disconnectMu.Lock()
	defer s.disconnectMu.Unlock()

	connections, err := s.Connections(usrSeq)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}
	if !connections.HasProvider(provider) {
		return model.SocialDisconnectResult{
			Status:      model.SocialDisconnectNotConnected,
			Connections: connections,
		}, nil
	}
	if !connections.HasAlternativeTo(provider) {
		return model.SocialDisconnectResult{}, ErrLastLoginMethod
	}
	if err := s.disconnect(ctx, usrSeq, provider, revocationActionDisconnect); err != nil {
		return model.SocialDisconnectResult{}, err
	}
	updatedConnections, err := s.Connections(usrSeq)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}
	return model.SocialDisconnectResult{
		Status:      model.SocialDisconnectCompleted,
		Connections: updatedConnections,
	}, nil
}

func (s *SocialAccountLifecycleService) DeleteAccount(ctx context.Context, usrSeq int) (AccountDeletionResult, error) {
	if err := s.auth.repo.UpdateMemberStatus(usrSeq, "AAA"); err != nil {
		return AccountDeletionResult{}, err
	}
	result := AccountDeletionResult{}
	if err := s.auth.repo.RevokeMobileRefreshTokensByUser(usrSeq); err != nil {
		result.RevocationPending = true
	}
	if err := s.auth.repo.DeleteLegacySessionsByUser(usrSeq); err != nil {
		result.RevocationPending = true
	}
	providers, err := s.auth.repo.ListSocialProviders(usrSeq)
	if err != nil {
		result.RevocationPending = true
		return result, nil
	}
	for _, rawProvider := range providers {
		provider := model.SocialProvider(rawProvider)
		if err := s.disconnect(ctx, usrSeq, provider, revocationActionDelete); err != nil {
			result.RevocationPending = true
		}
	}
	return result, nil
}

func (s *SocialAccountLifecycleService) ApplyAppleNotification(notification AppleServerNotification) error {
	user, err := s.auth.FindMemberBySocialID(string(model.SocialProviderApple), notification.Subject)
	if err != nil || user == nil {
		return err
	}
	switch notification.Type {
	case "consent-revoked":
		if err := s.auth.repo.RevokeMobileRefreshTokensByUser(user.USRSeq); err != nil {
			return err
		}
		if err := s.auth.repo.DeleteLegacySessionsByUser(user.USRSeq); err != nil {
			return err
		}
		return s.auth.repo.DeleteSocialConnection(user.USRSeq, string(model.SocialProviderApple))
	case "account-deleted":
		if err := s.auth.repo.UpdateMemberStatus(user.USRSeq, "AAA"); err != nil {
			return err
		}
		if err := s.auth.repo.RevokeMobileRefreshTokensByUser(user.USRSeq); err != nil {
			return err
		}
		if err := s.auth.repo.DeleteLegacySessionsByUser(user.USRSeq); err != nil {
			return err
		}
		return s.auth.repo.DeleteSocialConnection(user.USRSeq, string(model.SocialProviderApple))
	case "email-disabled":
		return s.auth.repo.UpdateSocialProviderState(user.USRSeq, string(model.SocialProviderApple), "ACTIVE", false)
	case "email-enabled":
		return s.auth.repo.UpdateSocialProviderState(user.USRSeq, string(model.SocialProviderApple), "ACTIVE", true)
	default:
		return errors.New("unsupported apple notification event")
	}
}

func (s *SocialAccountLifecycleService) disconnect(
	ctx context.Context,
	usrSeq int,
	provider model.SocialProvider,
	action string,
) error {
	if !provider.Valid() {
		return errors.New("unsupported social provider")
	}
	credential, err := s.loadCredential(usrSeq, provider)
	if err != nil {
		_ = s.auth.repo.EnqueueSocialRevocation(usrSeq, string(provider), action, err)
		return err
	}
	if credential == "" {
		err := errors.New("provider revocation credential is unavailable")
		_ = s.auth.repo.EnqueueSocialRevocation(usrSeq, string(provider), action, err)
		return err
	}
	if err := s.revoke(ctx, provider, credential); err != nil {
		_ = s.auth.repo.EnqueueSocialRevocation(usrSeq, string(provider), action, err)
		return err
	}
	return s.auth.repo.DeleteSocialConnection(usrSeq, string(provider))
}

func (s *SocialAccountLifecycleService) loadCredential(usrSeq int, provider model.SocialProvider) (string, error) {
	if s.vaultErr != nil {
		return "", s.vaultErr
	}
	encrypted, err := s.auth.repo.GetSocialCredential(usrSeq, string(provider))
	if err != nil || encrypted == "" {
		return "", err
	}
	return s.vault.Decrypt(encrypted)
}

func (s *SocialAccountLifecycleService) revoke(ctx context.Context, provider model.SocialProvider, credential string) error {
	switch provider {
	case model.SocialProviderKakao:
		return s.auth.UnlinkKakaoToken(ctx, credential)
	case model.SocialProviderApple:
		return s.apple.RevokeToken(ctx, credential)
	default:
		return errors.New("unsupported social provider")
	}
}
