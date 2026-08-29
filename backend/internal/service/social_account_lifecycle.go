package service

import (
	"context"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

const (
	revocationActionDisconnect = "DISCONNECT"
)

var ErrLastLoginMethod = repository.ErrLastLoginMethod

type SocialAccountLifecycleService struct {
	auth     *AuthService
	apple    *AppleIdentityVerifier
	vault    *SocialCredentialVault
	vaultErr error
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
	connections, phase, err := s.auth.repo.ReserveSocialDisconnect(usrSeq, provider)
	if err != nil {
		return model.SocialDisconnectResult{}, err
	}
	if phase == repository.SocialDisconnectNotConnected {
		return model.SocialDisconnectResult{
			Status:      model.SocialDisconnectNotConnected,
			Connections: connections,
		}, nil
	}
	if phase == repository.SocialDisconnectFinalizePending {
		if err := s.auth.repo.DeleteSocialConnection(usrSeq, string(provider)); err != nil {
			return model.SocialDisconnectResult{}, err
		}
	} else if err := s.disconnect(ctx, usrSeq, provider, revocationActionDisconnect, phase == repository.SocialDisconnectRevokeFresh); err != nil {
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

func (s *SocialAccountLifecycleService) DeleteAccount(_ context.Context, usrSeq int) (AccountDeletionResult, error) {
	err := s.auth.repo.AnonymizeAccountForDeletion(usrSeq)
	return AccountDeletionResult{}, err
}

func (s *SocialAccountLifecycleService) ApplyAppleNotification(notification AppleServerNotification) error {
	user, err := s.auth.FindMemberBySocialID(string(model.SocialProviderApple), notification.Subject)
	if err != nil || user == nil {
		return err
	}
	switch notification.Type {
	case "consent-revoked":
		if err := s.auth.repo.RevokeAllMobileSessions(user.USRSeq); err != nil {
			return err
		}
		if err := s.auth.repo.DeleteLegacySessionsByUser(user.USRSeq); err != nil {
			return err
		}
		return s.auth.repo.DeleteSocialConnection(user.USRSeq, string(model.SocialProviderApple))
	case "account-deleted":
		_, err := s.DeleteAccount(context.Background(), user.USRSeq)
		return err
	case "email-disabled":
		return s.auth.repo.UpdateSocialProviderEmailEnabled(user.USRSeq, string(model.SocialProviderApple), false)
	case "email-enabled":
		return s.auth.repo.UpdateSocialProviderEmailEnabled(user.USRSeq, string(model.SocialProviderApple), true)
	default:
		return errors.New("unsupported apple notification event")
	}
}

func (s *SocialAccountLifecycleService) disconnect(
	ctx context.Context,
	usrSeq int,
	provider model.SocialProvider,
	action string,
	restoreConnectionOnFailure bool,
) error {
	if !provider.Valid() {
		return errors.New("unsupported social provider")
	}
	credential, err := s.loadCredential(usrSeq, provider)
	if err != nil {
		return s.recordRevocationFailure(usrSeq, provider, action, err, restoreConnectionOnFailure)
	}
	if credential == "" {
		err := errors.New("provider revocation credential is unavailable")
		return s.recordRevocationFailure(usrSeq, provider, action, err, restoreConnectionOnFailure)
	}
	if err := s.revoke(ctx, provider, credential); err != nil {
		return s.recordRevocationFailure(usrSeq, provider, action, err, restoreConnectionOnFailure)
	}
	if action == revocationActionDisconnect {
		if err := s.auth.repo.MarkSocialDisconnectRevoked(usrSeq, string(provider)); err != nil {
			return err
		}
	}
	return s.auth.repo.DeleteSocialConnection(usrSeq, string(provider))
}

func (s *SocialAccountLifecycleService) recordRevocationFailure(usrSeq int, provider model.SocialProvider, action string, failure error, restoreConnection bool) error {
	if action != revocationActionDisconnect {
		return errors.Join(failure, s.auth.repo.EnqueueSocialRevocation(usrSeq, string(provider), action, failure))
	}
	releaseErr := s.auth.repo.RecordSocialDisconnectFailure(usrSeq, string(provider), failure, restoreConnection)
	return errors.Join(failure, releaseErr)
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
