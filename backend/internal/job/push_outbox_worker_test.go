package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

type fakePushOutboxWorkerStore struct {
	jobs       map[int]*repository.PushOutboxJob
	claimIDs   []int
	recovered  int64
	resetCalls int
	retries    []int
	dead       []int
	sent       []int
}

func newFakePushOutboxWorkerStore(jobs ...repository.PushOutboxJob) *fakePushOutboxWorkerStore {
	store := &fakePushOutboxWorkerStore{jobs: map[int]*repository.PushOutboxJob{}}
	for _, job := range jobs {
		copied := job
		store.jobs[job.POSeq] = &copied
		store.claimIDs = append(store.claimIDs, job.POSeq)
	}
	return store
}

func (f *fakePushOutboxWorkerStore) ClaimDue(_ context.Context, _ int) ([]repository.PushOutboxJob, error) {
	jobs := make([]repository.PushOutboxJob, 0, len(f.claimIDs))
	for _, id := range f.claimIDs {
		jobs = append(jobs, *f.jobs[id])
	}
	return jobs, nil
}

func (f *fakePushOutboxWorkerStore) MarkSent(_ context.Context, poSeq int) error {
	f.jobs[poSeq].Status = repository.PushOutboxStatusSent
	f.sent = append(f.sent, poSeq)
	return nil
}

func (f *fakePushOutboxWorkerStore) MarkRetryScheduled(_ context.Context, poSeq int, nextAttemptAt time.Time, errorCode string, errorMessage string) error {
	job := f.jobs[poSeq]
	job.Status = repository.PushOutboxStatusFailed
	job.AttemptCount++
	job.NextAttemptAt = nextAttemptAt
	job.LastErrorCode = sql.NullString{String: errorCode, Valid: errorCode != ""}
	job.LastErrorMessage = sql.NullString{String: errorMessage, Valid: errorMessage != ""}
	f.retries = append(f.retries, poSeq)
	return nil
}

func (f *fakePushOutboxWorkerStore) MarkDead(_ context.Context, poSeq int, errorCode string, errorMessage string) error {
	job := f.jobs[poSeq]
	job.Status = repository.PushOutboxStatusDead
	job.AttemptCount++
	job.LastErrorCode = sql.NullString{String: errorCode, Valid: errorCode != ""}
	job.LastErrorMessage = sql.NullString{String: errorMessage, Valid: errorMessage != ""}
	f.dead = append(f.dead, poSeq)
	return nil
}

func (f *fakePushOutboxWorkerStore) ResetStuckProcessing(_ context.Context, _ time.Duration) (int64, error) {
	f.resetCalls++
	return f.recovered, nil
}

type fakeWorkerPushProvider struct {
	err  error
	sent []service.PushNotification
}

func (f *fakeWorkerPushProvider) SendPush(_ context.Context, notification service.PushNotification) error {
	f.sent = append(f.sent, notification)
	return f.err
}

type fakeWorkerTokenRevoker struct {
	revoked []string
}

func (f *fakeWorkerTokenRevoker) RevokeToken(deviceToken string) error {
	f.revoked = append(f.revoked, deviceToken)
	return nil
}

type fakeWorkerPreferenceReader struct {
	preferences map[int]model.PushPreferences
	err         error
}

func (f *fakeWorkerPreferenceReader) GetPreferences(usrSeq int) (model.PushPreferences, error) {
	if f.err != nil {
		return model.PushPreferences{}, f.err
	}
	if f.preferences == nil {
		return model.DefaultPushPreferences(), nil
	}
	if preferences, ok := f.preferences[usrSeq]; ok {
		return preferences, nil
	}
	return model.DefaultPushPreferences(), nil
}

func TestPushOutboxWorkerSuccessMarksSent(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	provider := &fakeWorkerPushProvider{}
	worker := newTestPushOutboxWorker(store, &fakeWorkerTokenRevoker{}, provider)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(provider.sent) != 1 {
		t.Fatalf("expected provider called once, got %d", len(provider.sent))
	}
	if store.jobs[100].Status != repository.PushOutboxStatusSent || len(store.sent) != 1 {
		t.Fatalf("expected job sent, got %#v", store.jobs[100])
	}
}

func TestPushOutboxWorkerInvalidTokenRevokesAndMarksDead(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	revoker := &fakeWorkerTokenRevoker{}
	provider := &fakeWorkerPushProvider{err: &service.APNsResponseError{StatusCode: 400, Reason: "BadDeviceToken"}}
	worker := newTestPushOutboxWorker(store, revoker, provider)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(revoker.revoked) != 1 || revoker.revoked[0] != "token-1" {
		t.Fatalf("expected token revoked, got %#v", revoker.revoked)
	}
	if store.jobs[100].Status != repository.PushOutboxStatusDead || store.jobs[100].LastErrorCode.String != "BadDeviceToken" {
		t.Fatalf("expected dead invalid token job, got %#v", store.jobs[100])
	}
}

func TestPushOutboxWorkerTransientErrorSchedulesRetry(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	provider := &fakeWorkerPushProvider{err: &service.APNsResponseError{StatusCode: 429, Reason: "TooManyRequests"}}
	worker := newTestPushOutboxWorker(store, &fakeWorkerTokenRevoker{}, provider)
	worker.now = func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) }

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	got := store.jobs[100]
	if got.Status != repository.PushOutboxStatusFailed || got.AttemptCount != 1 || len(store.retries) != 1 {
		t.Fatalf("expected retry scheduled, got %#v", got)
	}
	if !got.NextAttemptAt.Equal(worker.now().Add(worker.cfg.BaseBackoff)) {
		t.Fatalf("unexpected retry time: %s", got.NextAttemptAt)
	}
}

func TestPushOutboxWorkerMaxAttemptsMarksDead(t *testing.T) {
	job := makeWorkerOutboxJob(100, 2)
	store := newFakePushOutboxWorkerStore(job)
	provider := &fakeWorkerPushProvider{err: &service.APNsResponseError{StatusCode: 503, Reason: "ServiceUnavailable"}}
	worker := newTestPushOutboxWorker(store, &fakeWorkerTokenRevoker{}, provider)
	worker.cfg.MaxAttempts = 3

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if store.jobs[100].Status != repository.PushOutboxStatusDead || len(store.dead) != 1 {
		t.Fatalf("expected job dead after max attempts, got %#v", store.jobs[100])
	}
}

func TestPushOutboxWorkerRecoversStuckProcessing(t *testing.T) {
	store := newFakePushOutboxWorkerStore()
	store.recovered = 2
	worker := newTestPushOutboxWorker(store, &fakeWorkerTokenRevoker{}, &fakeWorkerPushProvider{})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if store.resetCalls != 1 {
		t.Fatalf("expected reset called once, got %d", store.resetCalls)
	}
}

func TestPushOutboxWorkerUnknownErrorMarksDead(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	provider := &fakeWorkerPushProvider{err: errors.New("apns config is incomplete")}
	worker := newTestPushOutboxWorker(store, &fakeWorkerTokenRevoker{}, provider)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if store.jobs[100].Status != repository.PushOutboxStatusDead {
		t.Fatalf("expected unknown error dead, got %#v", store.jobs[100])
	}
}

func TestPushOutboxWorkerSkipsDisabledMessagePreference(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	preferences := &fakeWorkerPreferenceReader{
		preferences: map[int]model.PushPreferences{
			67890: {NoticeEnabled: true, MessageEnabled: false},
		},
	}
	provider := &fakeWorkerPushProvider{}
	worker := newTestPushOutboxWorkerWithPreferences(store, &fakeWorkerTokenRevoker{}, preferences, provider)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(provider.sent) != 0 {
		t.Fatalf("expected provider not called, got %d sends", len(provider.sent))
	}
	if store.jobs[100].Status != repository.PushOutboxStatusDead ||
		store.jobs[100].LastErrorCode.String != "PUSH_DISABLED_BY_USER" {
		t.Fatalf("expected disabled job dead, got %#v", store.jobs[100])
	}
}

func TestPushOutboxWorkerPreferenceLookupFailureRetries(t *testing.T) {
	job := makeWorkerOutboxJob(100, 0)
	store := newFakePushOutboxWorkerStore(job)
	preferences := &fakeWorkerPreferenceReader{err: errors.New("db down")}
	provider := &fakeWorkerPushProvider{}
	worker := newTestPushOutboxWorkerWithPreferences(store, &fakeWorkerTokenRevoker{}, preferences, provider)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(provider.sent) != 0 {
		t.Fatalf("expected provider not called, got %d sends", len(provider.sent))
	}
	if store.jobs[100].Status != repository.PushOutboxStatusFailed ||
		store.jobs[100].LastErrorCode.String != "PREFERENCE_LOOKUP_FAILED" {
		t.Fatalf("expected preference failure retry, got %#v", store.jobs[100])
	}
}

func newTestPushOutboxWorker(store *fakePushOutboxWorkerStore, revoker *fakeWorkerTokenRevoker, provider *fakeWorkerPushProvider) *PushOutboxWorker {
	return newTestPushOutboxWorkerWithPreferences(store, revoker, nil, provider)
}

func newTestPushOutboxWorkerWithPreferences(
	store *fakePushOutboxWorkerStore,
	revoker *fakeWorkerTokenRevoker,
	preferences *fakeWorkerPreferenceReader,
	provider *fakeWorkerPushProvider,
) *PushOutboxWorker {
	var preferenceReader PushPreferenceReader
	if preferences != nil {
		preferenceReader = preferences
	}
	return NewPushOutboxWorker(store, revoker, preferenceReader, provider, PushOutboxWorkerConfig{
		BatchSize:       10,
		PollInterval:    time.Hour,
		MaxAttempts:     3,
		BaseBackoff:     time.Minute,
		MaxBackoff:      10 * time.Minute,
		RecoveryTimeout: 5 * time.Minute,
		RequestTimeout:  time.Second,
	}, zerolog.Nop())
}

func makeWorkerOutboxJob(poSeq int, attemptCount int) repository.PushOutboxJob {
	payload := service.BuildMessageNewPushPayload(777, 12345, 67890, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))
	payloadJSON, err := json.Marshal(payload.CustomPayload())
	if err != nil {
		panic(err)
	}
	return repository.PushOutboxJob{
		POSeq:           poSeq,
		EventType:       service.PushEventMessageNew,
		EventID:         "message.new:777:67890",
		UsrSeq:          67890,
		MDTSeq:          1,
		DeviceToken:     "token-1",
		APNsEnvironment: "sandbox",
		BundleID:        sql.NullString{String: "com.daeil.dflhsafv2", Valid: true},
		Title:           "새 쪽지가 도착했습니다",
		Body:            "새로운 쪽지가 도착했습니다.",
		PayloadJSON:     string(payloadJSON),
		Status:          repository.PushOutboxStatusProcessing,
		AttemptCount:    attemptCount,
		NextAttemptAt:   time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		CreatedAt:       time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	}
}
