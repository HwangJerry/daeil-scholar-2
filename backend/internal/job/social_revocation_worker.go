// social_revocation_worker.go — Background worker draining
// ALUMNI_SOCIAL_REVOCATION_OUTBOX and calling out to the social provider
// (Kakao unlink, Apple revoke) so a disconnect/account-deletion actually
// revokes the upstream token instead of only queuing an outbox row.
//
// The synchronous disconnect/account-deletion paths (auth_repo.go's
// ReserveSocialDisconnect+disconnect, account_deletion_repo.go's
// AnonymizeAccountForDeletion) already attempt the provider call inline and
// only fall back to this outbox on failure - see
// SocialAccountLifecycleService.recordRevocationFailure. This worker is the
// retry path for those failures, plus the sole processor for ACCOUNT_DELETE
// entries once the synchronous attempt has failed once.
package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

const (
	socialRevocationBatchSize  = 20
	socialRevocationMaxAttempt = 10
	socialRevocationMaxBackoff = time.Hour
	// socialRevocationClaimStaleAfter bounds how long a claimed row is
	// considered "owned" by the worker that claimed it before another worker
	// (or this same worker, after a restart) is allowed to re-claim it. Must
	// comfortably exceed the time to process a full batch (network calls to
	// Kakao/Apple included), so a slow-but-alive worker never has a row
	// stolen out from under it mid-processing.
	socialRevocationClaimStaleAfter = 5 * time.Minute

	socialRevocationActionDisconnect    = "DISCONNECT"
	socialRevocationActionAccountDelete = "ACCOUNT_DELETE"
)

// socialRevocationRepository is the subset of *repository.AuthRepository this
// worker depends on, narrowed to an interface for testability.
type socialRevocationRepository interface {
	ClaimDueSocialRevocations(claimToken string, staleAfter time.Duration, limit int) ([]model.SocialRevocationOutboxEntry, error)
	MarkSocialRevocationRevoked(outboxID int64) error
	MarkSocialRevocationSucceeded(outboxID int64) error
	MarkSocialRevocationFailed(outboxID int64, errMsg string, newAttemptCount int, maxAttempts int, nextAttempt time.Time, retryStatus string) error
	FinalizeSocialDisconnect(usrSeq int, provider string) error
	CompleteAccountDeletionRevocation(usrSeq int, provider string) error
	GetSocialCredential(usrSeq int, provider string) (string, error)
}

// socialCredentialVault is the subset of *service.SocialCredentialVault this
// worker depends on.
type socialCredentialVault interface {
	Decrypt(value string) (string, error)
}

// socialTokenRevoker unlinks a Kakao access token. Satisfied by
// *service.AuthService.
type socialTokenRevoker interface {
	UnlinkKakaoToken(ctx context.Context, accessToken string) error
}

// appleTokenRevoker revokes an Apple refresh token. Satisfied by
// *service.AppleIdentityVerifier.
type appleTokenRevoker interface {
	RevokeToken(ctx context.Context, token string) error
}

// SocialRevocationWorker periodically drains ALUMNI_SOCIAL_REVOCATION_OUTBOX,
// resolving the stored credential for each pending entry and calling the
// relevant provider's revoke/unlink API.
type SocialRevocationWorker struct {
	repo       socialRevocationRepository
	kakao      socialTokenRevoker
	apple      appleTokenRevoker
	vault      socialCredentialVault
	vaultErr   error
	interval   time.Duration
	logger     zerolog.Logger
	cancel     context.CancelFunc
	done       chan struct{}
	claimToken string
}

// NewSocialRevocationWorker creates a SocialRevocationWorker. vault/vaultErr
// mirror the credential vault construction used elsewhere (e.g.
// SocialAccountLifecycleService) so the worker can decrypt stored provider
// credentials the same way they were encrypted when saved.
func NewSocialRevocationWorker(
	repo socialRevocationRepository,
	kakao socialTokenRevoker,
	apple appleTokenRevoker,
	vault socialCredentialVault,
	vaultErr error,
	interval time.Duration,
	logger zerolog.Logger,
) *SocialRevocationWorker {
	return &SocialRevocationWorker{
		repo:       repo,
		kakao:      kakao,
		apple:      apple,
		vault:      vault,
		vaultErr:   vaultErr,
		interval:   interval,
		logger:     logger,
		claimToken: generateClaimToken(),
	}
}

// generateClaimToken returns a random identifier this worker instance uses to
// mark which outbox rows it currently owns (see ClaimDueSocialRevocations).
// Generated once per worker instance/process, not per poll tick: every claim
// this process makes is distinguishable from any other process's claims, and
// a fresh process always gets a fresh token, so it can never mistake a prior
// process's abandoned claim for its own.
func generateClaimToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely (crypto/rand failure); fall back to a
		// zero-value token rather than panicking - worst case this worker's
		// claims collide with a hypothetical other zero-token worker, which
		// is no worse than having no claiming at all.
		return "fallback-0000000000000000"
	}
	return hex.EncodeToString(b)
}

// Start begins polling the outbox on a ticker in a background goroutine.
func (w *SocialRevocationWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		defer func() {
			if r := recover(); r != nil {
				w.logger.Error().Interface("panic", r).Msg("social revocation worker panicked")
			}
		}()
		w.logger.Info().Dur("interval", w.interval).Msg("social revocation worker started")
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info().Msg("social revocation worker stopped")
				return
			case <-ticker.C:
				w.processDue(ctx)
			}
		}
	}()
}

// Stop signals the background goroutine to exit and waits for it to finish
// its current tick, if any.
func (w *SocialRevocationWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	if w.done != nil {
		<-w.done
	}
}

func (w *SocialRevocationWorker) processDue(ctx context.Context) {
	entries, err := w.repo.ClaimDueSocialRevocations(w.claimToken, socialRevocationClaimStaleAfter, socialRevocationBatchSize)
	if err != nil {
		w.logger.Error().Err(err).Msg("social revocation worker failed to claim due entries")
		return
	}
	for _, entry := range entries {
		w.processEntry(ctx, entry)
	}
}

func (w *SocialRevocationWorker) processEntry(ctx context.Context, entry model.SocialRevocationOutboxEntry) {
	// If a prior attempt already got the upstream provider to revoke the
	// credential (STATUS=REVOKED), do NOT call the provider again: Kakao/Apple
	// reject a second unlink/revoke of an already-revoked credential, which
	// would permanently strand the row if we retried the full flow from
	// scratch. Only retry local finalization in that case.
	alreadyRevoked := entry.Status == "REVOKED"

	if !alreadyRevoked {
		if err := w.revoke(ctx, entry); err != nil {
			// The provider was never successfully contacted, so retrying the
			// full flow (including the provider call) is correct.
			w.recordFailure(entry, err, "PENDING")
			return
		}
		// Upstream revoke just succeeded. Persist that fact durably BEFORE
		// attempting local finalization: if finalization or even this very
		// write fails/crashes, a retry must never call the provider again.
		if err := w.repo.MarkSocialRevocationRevoked(entry.OutboxID); err != nil {
			w.recordFailure(entry, err, "REVOKED")
			return
		}
	}

	if err := w.finalize(entry); err != nil {
		w.recordFailure(entry, err, "REVOKED")
		return
	}
	if err := w.repo.MarkSocialRevocationSucceeded(entry.OutboxID); err != nil {
		w.logger.Error().Err(err).Int64("outboxId", entry.OutboxID).Msg("failed to release social revocation claim")
		return
	}
	w.logger.Info().
		Int64("outboxId", entry.OutboxID).
		Int("usrSeq", entry.USRSeq).
		Str("provider", entry.Provider).
		Str("action", entry.Action).
		Msg("social revocation succeeded")
}

// finalize performs the same local completion the synchronous disconnect/
// account-deletion paths perform after a successful revoke, dispatched by
// entry.Action. Both underlying repository methods mark the outbox row
// DELIVERED themselves as part of the same transaction.
func (w *SocialRevocationWorker) finalize(entry model.SocialRevocationOutboxEntry) error {
	switch entry.Action {
	case socialRevocationActionDisconnect:
		return w.repo.FinalizeSocialDisconnect(entry.USRSeq, entry.Provider)
	case socialRevocationActionAccountDelete:
		return w.repo.CompleteAccountDeletionRevocation(entry.USRSeq, entry.Provider)
	default:
		return errors.New("unsupported social revocation action: " + entry.Action)
	}
}

// recordFailure marks entry as failed and due for retry at a backoff-delayed
// time, unless the attempt cap has been reached (in which case it becomes
// terminal FAILED). retryStatus controls whether the retry re-attempts the
// upstream provider call ("PENDING") or only local finalization ("REVOKED") -
// see MarkSocialRevocationFailed for why this distinction matters.
func (w *SocialRevocationWorker) recordFailure(entry model.SocialRevocationOutboxEntry, err error, retryStatus string) {
	attemptCount := entry.AttemptCount + 1
	nextAttempt := time.Now().Add(backoffDuration(attemptCount))
	if markErr := w.repo.MarkSocialRevocationFailed(entry.OutboxID, err.Error(), attemptCount, socialRevocationMaxAttempt, nextAttempt, retryStatus); markErr != nil {
		w.logger.Error().Err(markErr).Int64("outboxId", entry.OutboxID).Msg("failed to record social revocation failure")
		return
	}

	logEvent := w.logger.Warn()
	if attemptCount >= socialRevocationMaxAttempt {
		logEvent = w.logger.Error()
	}
	logEvent.
		Err(err).
		Int64("outboxId", entry.OutboxID).
		Int("usrSeq", entry.USRSeq).
		Str("provider", entry.Provider).
		Str("action", entry.Action).
		Str("retryStatus", retryStatus).
		Int("attemptCount", attemptCount).
		Msg("social revocation failed")
}

// revoke resolves the stored credential for the outbox entry's user+provider
// and calls the provider's revoke/unlink API, mirroring the decrypt-and-call
// pattern used by SocialAccountLifecycleService when a credential is
// available at disconnect time.
func (w *SocialRevocationWorker) revoke(ctx context.Context, entry model.SocialRevocationOutboxEntry) error {
	provider := model.SocialProvider(entry.Provider)
	if !provider.Valid() {
		return errors.New("unsupported social provider")
	}

	credential, err := w.loadCredential(entry.USRSeq, provider)
	if err != nil {
		return err
	}
	if credential == "" {
		return errors.New("no stored credential to revoke")
	}

	switch provider {
	case model.SocialProviderKakao:
		return w.kakao.UnlinkKakaoToken(ctx, credential)
	case model.SocialProviderApple:
		if w.apple == nil {
			return errors.New("apple revocation not configured")
		}
		return w.apple.RevokeToken(ctx, credential)
	default:
		return errors.New("unsupported social provider")
	}
}

func (w *SocialRevocationWorker) loadCredential(usrSeq int, provider model.SocialProvider) (string, error) {
	if w.vaultErr != nil {
		return "", w.vaultErr
	}
	encrypted, err := w.repo.GetSocialCredential(usrSeq, string(provider))
	if err != nil || encrypted == "" {
		return "", err
	}
	return w.vault.Decrypt(encrypted)
}

// backoffDuration returns an exponential-ish backoff capped at
// socialRevocationMaxBackoff: min(2^attemptCount minutes, 1 hour).
func backoffDuration(attemptCount int) time.Duration {
	minutes := math.Pow(2, float64(attemptCount))
	d := time.Duration(minutes) * time.Minute
	if d > socialRevocationMaxBackoff || d <= 0 {
		return socialRevocationMaxBackoff
	}
	return d
}
