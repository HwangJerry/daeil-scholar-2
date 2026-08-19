// social_revocation_worker_test.go — Verifies the worker's revoke/finalize/
// retry state machine against fake repository and provider dependencies.
package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

type fakeSocialRevocationRepo struct {
	credential            string
	credentialErr         error
	revokedCalls          []int64
	revokedErr            error
	succeededCalls        []int64
	succeededErr          error
	failedCalls           []failedCall
	failedErr             error
	finalizeDisconnect    []int
	finalizeDisconnectErr error
	completeDeleteCalls   []int
	completeDeleteErr     error
}

type failedCall struct {
	outboxID     int64
	errMsg       string
	attemptCount int
	maxAttempts  int
	retryStatus  string
}

func (f *fakeSocialRevocationRepo) ClaimDueSocialRevocations(string, time.Duration, int) ([]model.SocialRevocationOutboxEntry, error) {
	return nil, nil // not exercised directly; processEntry is tested in isolation
}

func (f *fakeSocialRevocationRepo) MarkSocialRevocationRevoked(outboxID int64) error {
	f.revokedCalls = append(f.revokedCalls, outboxID)
	return f.revokedErr
}

func (f *fakeSocialRevocationRepo) MarkSocialRevocationSucceeded(outboxID int64) error {
	f.succeededCalls = append(f.succeededCalls, outboxID)
	return f.succeededErr
}

func (f *fakeSocialRevocationRepo) MarkSocialRevocationFailed(outboxID int64, errMsg string, attemptCount int, maxAttempts int, _ time.Time, retryStatus string) error {
	f.failedCalls = append(f.failedCalls, failedCall{outboxID, errMsg, attemptCount, maxAttempts, retryStatus})
	return f.failedErr
}

func (f *fakeSocialRevocationRepo) FinalizeSocialDisconnect(usrSeq int, _ string) error {
	f.finalizeDisconnect = append(f.finalizeDisconnect, usrSeq)
	return f.finalizeDisconnectErr
}

func (f *fakeSocialRevocationRepo) CompleteAccountDeletionRevocation(usrSeq int, _ string) error {
	f.completeDeleteCalls = append(f.completeDeleteCalls, usrSeq)
	return f.completeDeleteErr
}

func (f *fakeSocialRevocationRepo) GetSocialCredential(int, string) (string, error) {
	return f.credential, f.credentialErr
}

type fakeVault struct {
	decrypted string
	err       error
}

func (v *fakeVault) Decrypt(string) (string, error) { return v.decrypted, v.err }

type fakeKakao struct {
	calls int
	err   error
}

func (f *fakeKakao) UnlinkKakaoToken(context.Context, string) error {
	f.calls++
	return f.err
}

func newTestWorker(repo socialRevocationRepository, kakao socialTokenRevoker) *SocialRevocationWorker {
	return &SocialRevocationWorker{
		repo:       repo,
		kakao:      kakao,
		vault:      &fakeVault{decrypted: "plain-token"},
		interval:   time.Minute,
		logger:     zerolog.Nop(),
		claimToken: "test-token",
	}
}

func TestProcessEntryDisconnectSuccess(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "encrypted"}
	kakao := &fakeKakao{}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 1, USRSeq: 42, Provider: "KT", Action: socialRevocationActionDisconnect, Status: "PENDING"}
	w.processEntry(context.Background(), entry)

	if kakao.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", kakao.calls)
	}
	if len(repo.revokedCalls) != 1 || repo.revokedCalls[0] != 1 {
		t.Fatalf("expected MarkSocialRevocationRevoked(1), got %v", repo.revokedCalls)
	}
	if len(repo.finalizeDisconnect) != 1 || repo.finalizeDisconnect[0] != 42 {
		t.Fatalf("expected FinalizeSocialDisconnect(42), got %v", repo.finalizeDisconnect)
	}
	if len(repo.succeededCalls) != 1 {
		t.Fatalf("expected MarkSocialRevocationSucceeded called once, got %v", repo.succeededCalls)
	}
	if len(repo.failedCalls) != 0 {
		t.Fatalf("expected no failures, got %v", repo.failedCalls)
	}
}

func TestProcessEntryAccountDeleteSuccess(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "encrypted"}
	kakao := &fakeKakao{}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 2, USRSeq: 43, Provider: "KT", Action: socialRevocationActionAccountDelete, Status: "PENDING"}
	w.processEntry(context.Background(), entry)

	if len(repo.completeDeleteCalls) != 1 || repo.completeDeleteCalls[0] != 43 {
		t.Fatalf("expected CompleteAccountDeletionRevocation(43), got %v", repo.completeDeleteCalls)
	}
	if len(repo.finalizeDisconnect) != 0 {
		t.Fatalf("account-delete entries must not call FinalizeSocialDisconnect, got %v", repo.finalizeDisconnect)
	}
}

// TestProcessEntryAlreadyRevokedSkipsProviderCall proves an entry already
// checkpointed REVOKED never calls the provider again, even if a prior
// attempt's finalize step is what's being retried.
func TestProcessEntryAlreadyRevokedSkipsProviderCall(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "encrypted"}
	kakao := &fakeKakao{}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 3, USRSeq: 44, Provider: "KT", Action: socialRevocationActionDisconnect, Status: "REVOKED", AttemptCount: 1}
	w.processEntry(context.Background(), entry)

	if kakao.calls != 0 {
		t.Fatalf("expected provider not to be called for an already-REVOKED entry, got %d calls", kakao.calls)
	}
	if len(repo.revokedCalls) != 0 {
		t.Fatalf("checkpoint should not be re-written when already REVOKED, got %v", repo.revokedCalls)
	}
	if len(repo.finalizeDisconnect) != 1 {
		t.Fatalf("expected finalize to still run, got %v", repo.finalizeDisconnect)
	}
}

// TestProcessEntryProviderFailureRetriesAsPending proves a failed provider
// call is retried as PENDING (never having revoked anything yet).
func TestProcessEntryProviderFailureRetriesAsPending(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "encrypted"}
	kakao := &fakeKakao{err: errors.New("kakao unavailable")}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 4, USRSeq: 45, Provider: "KT", Action: socialRevocationActionDisconnect, Status: "PENDING"}
	w.processEntry(context.Background(), entry)

	if len(repo.failedCalls) != 1 {
		t.Fatalf("expected 1 failure record, got %v", repo.failedCalls)
	}
	if repo.failedCalls[0].retryStatus != "PENDING" {
		t.Fatalf("expected retryStatus=PENDING, got %q", repo.failedCalls[0].retryStatus)
	}
	if len(repo.revokedCalls) != 0 || len(repo.finalizeDisconnect) != 0 {
		t.Fatalf("no checkpoint or finalize should happen on provider failure")
	}
}

// TestProcessEntryFinalizeFailureRetriesAsRevoked proves a failure AFTER a
// successful provider call is retried as REVOKED, never re-calling the
// provider - this is the exact bug class the durable checkpoint exists to
// prevent.
func TestProcessEntryFinalizeFailureRetriesAsRevoked(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "encrypted", finalizeDisconnectErr: errors.New("db blip")}
	kakao := &fakeKakao{}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 5, USRSeq: 46, Provider: "KT", Action: socialRevocationActionDisconnect, Status: "PENDING"}
	w.processEntry(context.Background(), entry)

	if kakao.calls != 1 {
		t.Fatalf("expected provider to be called exactly once, got %d", kakao.calls)
	}
	if len(repo.revokedCalls) != 1 {
		t.Fatalf("expected the REVOKED checkpoint to be written before finalize, got %v", repo.revokedCalls)
	}
	if len(repo.failedCalls) != 1 || repo.failedCalls[0].retryStatus != "REVOKED" {
		t.Fatalf("expected a REVOKED-retry failure record, got %v", repo.failedCalls)
	}
}

// TestProcessEntryAttemptCapReachesFailedStatus proves recordFailure passes
// through the accumulating attempt count so the repository layer can
// terminate retries at the cap (behavior itself is repository-owned; this
// just checks the worker plumbs AttemptCount+1 through).
func TestProcessEntryAttemptCapReachesFailedStatus(t *testing.T) {
	repo := &fakeSocialRevocationRepo{credential: "", credentialErr: nil}
	kakao := &fakeKakao{}
	w := newTestWorker(repo, kakao)

	entry := model.SocialRevocationOutboxEntry{OutboxID: 6, USRSeq: 47, Provider: "KT", Action: socialRevocationActionDisconnect, Status: "PENDING", AttemptCount: 9}
	w.processEntry(context.Background(), entry)

	if len(repo.failedCalls) != 1 {
		t.Fatalf("expected 1 failure record, got %v", repo.failedCalls)
	}
	if repo.failedCalls[0].attemptCount != 10 {
		t.Fatalf("expected attemptCount=10 (9+1), got %d", repo.failedCalls[0].attemptCount)
	}
}
