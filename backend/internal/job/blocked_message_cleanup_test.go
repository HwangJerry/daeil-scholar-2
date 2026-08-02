package job

import (
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type blockedCleanupRepoStub struct {
	limits []int
	errors []error
	calls  chan struct{}
}

type blockingCleanupRepoStub struct {
	started chan struct{}
	release chan struct{}
}

func (s *blockingCleanupRepoStub) DeleteExpiredSuppressedMessages(int) (int64, error) {
	close(s.started)
	<-s.release
	return 0, nil
}

func (s *blockedCleanupRepoStub) DeleteExpiredSuppressedMessages(limit int) (int64, error) {
	s.limits = append(s.limits, limit)
	if s.calls != nil {
		s.calls <- struct{}{}
	}
	index := len(s.limits) - 1
	if index < len(s.errors) && s.errors[index] != nil {
		return 0, s.errors[index]
	}
	return 1, nil
}

func TestBlockedMessageCleanupRunUsesApprovedBatchAndRetriesAfterFailure(t *testing.T) {
	repo := &blockedCleanupRepoStub{errors: []error{errors.New("temporary database failure"), nil}}
	job := NewBlockedMessageCleanupJob(repo, zerolog.Nop())
	job.runOnce()
	job.runOnce()
	if len(repo.limits) != 2 {
		t.Fatalf("calls = %d, want 2", len(repo.limits))
	}
	if blockedMessageCleanupBatchSize != 100 {
		t.Fatalf("batch size = %d, want 100", blockedMessageCleanupBatchSize)
	}
	for _, limit := range repo.limits {
		if limit != blockedMessageCleanupBatchSize {
			t.Fatalf("limit = %d, want 100", limit)
		}
	}
}

func TestBlockedMessageCleanupStartsImmediatelyAndStops(t *testing.T) {
	repo := &blockedCleanupRepoStub{calls: make(chan struct{}, 2)}
	job := NewBlockedMessageCleanupJob(repo, zerolog.Nop())
	job.interval = time.Hour
	job.Start()
	defer job.Stop()
	select {
	case <-repo.calls:
	case <-time.After(time.Second):
		t.Fatal("startup cleanup was not executed")
	}
}

func TestBlockedMessageCleanupStopWaitsForInFlightDelete(t *testing.T) {
	repo := &blockingCleanupRepoStub{started: make(chan struct{}), release: make(chan struct{})}
	job := NewBlockedMessageCleanupJob(repo, zerolog.Nop())
	job.Start()
	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not start")
	}

	stopped := make(chan struct{})
	go func() {
		job.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned before in-flight delete completed")
	case <-time.After(20 * time.Millisecond):
	}

	close(repo.release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after in-flight delete completed")
	}
}
