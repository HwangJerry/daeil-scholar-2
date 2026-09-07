// account_erasure.go — Retryable automatic deletion while preserving operator takeover.
package service

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

type AutomaticErasureStore interface {
	PrepareErasure(model.ErasureWork, func(model.DonationRetentionDecision) error) error
	ErasureLock(context.Context) (func(), bool, error)
	ErasureBatch(context.Context) ([]model.ErasureWork, error)
	ErasureActive(int64) (bool, error)
	Start(int64, int) error
	ErasureRetry(int64, string) error
	ErasureExternalSubject(model.ErasureWork) (model.ErasureExternalSubject, error)
	RecordExternalErasure(int64, string) error
	SaveErasureContext(int64, []byte) error
	LoadErasureContext(int64) ([]byte, error)
	ReceiptWork(int64) (model.ErasureReceiptWork, error)
	ErasureTargets(int64) ([]model.ErasureTarget, error)
	BeginErasureTargets(int64) error
	RecordErasureTargets(int64, []model.ErasureTarget) error
	EraseDatabase(model.ErasureWork, func([]byte) ([]byte, error), func(model.DonationRetentionDecision) error) error
	ErasureFiles(int64) ([]model.ErasureFile, error)
	ErasureFileDone(int64) error
	EraseFileIfUnreferenced(model.ErasureWork, model.ErasureFile, func(string) error) error
	FinishAutomaticErasure(model.ErasureWork) error
}
type ExternalErasureProcessor interface {
	Erase(context.Context, model.ErasureExternalSubject) (string, error)
}
type ErasureFileStorage interface{ EraseURL(string) error }
type AutomaticErasureService struct {
	ContextCipher   *ErasureContextCipher
	Store           AutomaticErasureStore
	External        ExternalErasureProcessor
	Files           ErasureFileStorage
	Seal            func([]byte) ([]byte, error)
	InvalidateCache func()
}

func (s *AutomaticErasureService) RunOnce(ctx context.Context) error {
	release, locked, err := s.Store.ErasureLock(ctx)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer release()
	batch, err := s.Store.ErasureBatch(ctx)
	if err != nil {
		return err
	}
	for _, w := range batch {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = s.process(ctx, w)
		if err != nil {
			code := "ERASURE_RETRY_REQUIRED"
			var blocked *model.ErasureBlocked
			if errors.As(err, &blocked) {
				code = blocked.Code
			}
			// Never persist raw SQL, paths, provider responses or personal data.
			if retryErr := s.Store.ErasureRetry(w.RequestID, code); retryErr != nil {
				return retryErr
			}
		}
	}
	return nil
}
func (s *AutomaticErasureService) process(ctx context.Context, w model.ErasureWork) error {
	active, err := s.Store.ErasureActive(w.RequestID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	if err = s.Store.Start(w.RequestID, 0); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	active, err = s.Store.ErasureActive(w.RequestID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	if w.Stage != "database_erased" {
		if err = s.Store.PrepareErasure(w, ValidateDonationRetention); err != nil {
			return err
		}
		if err = s.preserveContext(w); err != nil {
			return err
		}
		if err = s.Store.EraseDatabase(w, s.Seal, ValidateDonationRetention); err != nil {
			return err
		}
		if s.InvalidateCache != nil {
			s.InvalidateCache()
		}
	}
	for {
		active, err = s.Store.ErasureActive(w.RequestID)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}
		files, e := s.Store.ErasureFiles(w.RequestID)
		if e != nil {
			return e
		}
		if len(files) == 0 {
			break
		}
		for _, file := range files {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if s.Files == nil {
				return &model.ErasureBlocked{Code: "FILE_STORAGE_REQUIRED"}
			}
			if e = s.Store.EraseFileIfUnreferenced(w, file, s.Files.EraseURL); e != nil {
				return e
			}
		}
	}
	if err = s.processExternal(ctx, w); err != nil {
		return err
	}
	return s.Store.FinishAutomaticErasure(w)
}
