// account_erasure_workflow_test.go — Failures cannot become false completion or double erasure.
package service

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"testing"
)

type erasureStoreFake struct {
	work                     model.ErasureWork
	active                   bool
	databaseCalls, completed int
	files                    []model.ErasureFile
	retry                    string
	preparedError            error
	encrypted                []byte
	targets                  []model.ErasureTarget
}

func (f *erasureStoreFake) ErasureLock(context.Context) (func(), bool, error) {
	return func() {}, true, nil
}
func (f *erasureStoreFake) ErasureBatch(context.Context) ([]model.ErasureWork, error) {
	return []model.ErasureWork{f.work}, nil
}
func (f *erasureStoreFake) ErasureActive(int64) (bool, error)       { return f.active, nil }
func (f *erasureStoreFake) Start(int64, int) error                  { return nil }
func (f *erasureStoreFake) ErasureRetry(_ int64, code string) error { f.retry = code; return nil }
func (f *erasureStoreFake) PrepareErasure(model.ErasureWork, func(model.DonationRetentionDecision) error) error {
	return f.preparedError
}
func (f *erasureStoreFake) ErasureExternalSubject(w model.ErasureWork) (model.ErasureExternalSubject, error) {
	return model.ErasureExternalSubject{RequestID: w.RequestID, UserSeq: w.UserSeq}, nil
}
func (f *erasureStoreFake) RecordExternalErasure(_ int64, evidence string) error {
	f.work.ExternalEvidence = evidence
	return nil
}
func (f *erasureStoreFake) SaveErasureContext(_ int64, data []byte) error {
	f.encrypted = data
	return nil
}
func (f *erasureStoreFake) LoadErasureContext(int64) ([]byte, error) {
	if len(f.encrypted) == 0 {
		return nil, sql.ErrNoRows
	}
	return f.encrypted, nil
}
func testContextCipher(t *testing.T) *ErasureContextCipher {
	t.Helper()
	c, err := NewErasureContextCipher(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *erasureStoreFake) EraseDatabase(model.ErasureWork, func([]byte) ([]byte, error), func(model.DonationRetentionDecision) error) error {
	f.databaseCalls++
	f.work.Stage = "database_erased"
	return nil
}
func (f *erasureStoreFake) ErasureFiles(int64) ([]model.ErasureFile, error) { return f.files, nil }
func (f *erasureStoreFake) ErasureFileDone(int64) error                     { f.files = f.files[1:]; return nil }
func (f *erasureStoreFake) FinishAutomaticErasure(model.ErasureWork) error  { f.completed++; return nil }

type erasureExternalFake struct {
	err   error
	calls int
}

func (f *erasureExternalFake) Erase(context.Context, model.ErasureExternalSubject) (string, error) {
	f.calls++
	return "test-evidence", f.err
}

type erasureFilesFake struct {
	err   error
	calls int
}

func (f *erasureFilesFake) EraseURL(string) error { f.calls++; return f.err }

func TestAutomaticErasureResumesFilesAfterFailure(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42}, files: []model.ErasureFile{{ID: 1, URL: "/uploads/test.jpg"}}}
	external := &erasureExternalFake{}
	files := &erasureFilesFake{err: errors.New("private path must not be logged")}
	s := &AutomaticErasureService{ContextCipher: testContextCipher(t), Store: store, External: external, Files: files}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 0 || len(store.files) != 1 || store.databaseCalls != 1 || store.retry != "ERASURE_RETRY_REQUIRED" {
		t.Fatal("failure lost work or leaked details")
	}
	files.err = nil
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 1 || store.databaseCalls != 1 || external.calls != 1 || len(store.files) != 0 {
		t.Fatal("retry repeated database or external erasure")
	}
}

func TestManualTakeoverAndUnverifiedRetentionPreventAutomaticMutation(t *testing.T) {
	store := &erasureStoreFake{work: model.ErasureWork{RequestID: 1, UserSeq: 42}}
	external := &erasureExternalFake{}
	s := &AutomaticErasureService{ContextCipher: testContextCipher(t), Store: store, External: external}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if external.calls != 0 || store.databaseCalls != 0 {
		t.Fatal("manual request was processed")
	}
	store.active = true
	store.preparedError = &model.ErasureBlocked{Code: "DONATION_RETENTION_REVIEW_REQUIRED"}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if external.calls != 0 || store.databaseCalls != 0 || store.completed != 0 {
		t.Fatal("unreviewed financial records exposed to deletion")
	}
	store.preparedError = nil
	external.err = &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.databaseCalls != 1 || store.completed != 0 || store.retry != "EXTERNAL_ERASURE_PENDING" {
		t.Fatal("external failure counted as success")
	}
}

func TestExternalFailureResumesWithoutMemberRows(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42}}
	external := &erasureExternalFake{err: &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}}
	s := &AutomaticErasureService{Store: store, External: external, ContextCipher: testContextCipher(t)}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.databaseCalls != 1 || store.completed != 0 || len(store.encrypted) == 0 {
		t.Fatal("operational deletion blocked or work lost")
	}
	external.err = nil
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.databaseCalls != 1 || store.completed != 1 || external.calls != 2 {
		t.Fatal("resume repeated database erasure")
	}
}
func TestMissingContextKeyPreventsLosingExternalIdentifiers(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42}}
	s := &AutomaticErasureService{Store: store}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.databaseCalls != 0 || store.retry != "ERASURE_CONTEXT_KEY_REQUIRED" {
		t.Fatal("unrecoverable erasure allowed")
	}
}
func TestExpiredContextNeverCompletesOrRepeatsDatabaseErasure(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42, Stage: "database_erased"}}
	s := &AutomaticErasureService{Store: store, External: &erasureExternalFake{}, ContextCipher: testContextCipher(t)}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.databaseCalls != 0 || store.completed != 0 || store.retry != "ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED" {
		t.Fatal("expired context accepted")
	}
}

func (f *erasureStoreFake) ErasureTargets(int64) ([]model.ErasureTarget, error) {
	if f.targets == nil {
		for _, name := range model.ErasureTargetNames {
			f.targets = append(f.targets, model.ErasureTarget{Name: name, Status: "pending"})
		}
	}
	return f.targets, nil
}
func (f *erasureStoreFake) BeginErasureTargets(int64) error { return nil }
func (f *erasureStoreFake) RecordErasureTargets(_ int64, targets []model.ErasureTarget) error {
	for _, target := range targets {
		for i := range f.targets {
			if f.targets[i].Name == target.Name {
				f.targets[i] = target
			}
		}
	}
	return nil
}
