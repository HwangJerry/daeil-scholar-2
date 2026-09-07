// account_erasure_context.go — Persist and resume external work after operational deletion.
package service

import (
	"context"
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"reflect"
)

func (s *AutomaticErasureService) preserveContext(w *model.ErasureWork) error {
	if w.ExternalEvidence != "" {
		return nil
	}
	encrypted, loadErr := s.Store.LoadErasureContext(w.RequestID)
	if loadErr != nil && loadErr != sql.ErrNoRows {
		return loadErr
	}
	subject, err := s.Store.ErasureExternalSubject(*w)
	if err != nil {
		return err
	}
	w.ContextRetentions = subject.Retentions
	if loadErr == nil {
		previous, err := s.ContextCipher.Open(*w, encrypted)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(previous, subject) {
			return nil
		}
	}
	encrypted, err = s.ContextCipher.Seal(subject)
	if err != nil {
		return err
	}
	if loadErr == nil {
		return s.Store.RefreshErasureContext(w.RequestID, encrypted)
	}
	return s.Store.SaveErasureContext(w.RequestID, encrypted)
}

func (s *AutomaticErasureService) processExternal(ctx context.Context, w model.ErasureWork) error {
	if w.ExternalEvidence != "" {
		return nil
	}
	active, err := s.Store.ErasureActive(w.RequestID)
	if err != nil {
		return err
	}
	if !active {
		return sql.ErrNoRows
	}
	work, err := s.Store.ReceiptWork(w.RequestID)
	if err != nil {
		return err
	}
	if work.Status == "active" || work.Status == "unreviewed" {
		return &model.ErasureBlocked{Code: "RECEIPT_CONTACT_WORK_PENDING"}
	}
	targets, err := s.Store.ErasureTargets(w.RequestID)
	if err != nil {
		return err
	}
	verified := len(targets) == len(model.ErasureTargetNames)
	for _, target := range targets {
		verified = verified && target.Verified()
	}
	if verified {
		return s.Store.RecordExternalErasure(w.RequestID, "verified-per-storage: see target evidence")
	}
	if s.External == nil {
		return &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PROCESSOR_REQUIRED"}
	}
	encrypted, err := s.Store.LoadErasureContext(w.RequestID)
	if err == sql.ErrNoRows {
		return &model.ErasureBlocked{Code: "ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED"}
	}
	if err != nil {
		return err
	}
	subject, err := s.ContextCipher.Open(w, encrypted)
	if err != nil {
		return err
	}
	return s.erasePendingTargets(ctx, subject)
}
