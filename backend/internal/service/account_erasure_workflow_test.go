// account_erasure_workflow_test.go — Failures cannot become false completion or double erasure.
package service

import (
	"context"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"testing"
)

type erasureStoreFake struct {
	work                     model.ErasureWork
	active                   bool
	databaseCalls, completed int
	files                    []model.ErasureFile
	retry                    string
	preparedError            error
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
func (f *erasureStoreFake) RecordExternalErasure(int64, string) error { return nil }
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
	s := &AutomaticErasureService{Store: store, External: external, Files: files}
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
	s := &AutomaticErasureService{Store: store, External: external}
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
	if store.databaseCalls != 0 || store.completed != 0 || store.retry != "EXTERNAL_ERASURE_PENDING" {
		t.Fatal("external failure counted as success")
	}
}
