// account_erasure_targets.go — Validate per-target proof and resume only unfinished work.
package service

import (
	"context"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
)

type detailedErasureProcessor interface {
	EraseTargets(context.Context, model.ErasureExternalSubject) ([]model.ErasureTarget, error)
}

func validateExternalTargets(required []string, targets []model.ErasureTarget) ([]model.ErasureTarget, error) {
	blocked := &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	if len(required) == 0 {
		required = model.ErasureTargetNames
	}
	if len(targets) != len(required) {
		return nil, blocked
	}
	expected := map[string]bool{}
	for _, name := range required {
		expected[name] = true
	}
	for _, target := range targets {
		if !expected[target.Name] || len(target.Evidence) > 200 {
			return nil, blocked
		}
		delete(expected, target.Name)
		switch target.Status {
		case "pending", "manual", "failed":
			if target.Evidence != "" {
				return nil, blocked
			} // No unreviewed remote text persisted.
		case "complete", "not_applicable":
			if strings.TrimSpace(target.Evidence) == "" {
				return nil, blocked
			}
		default:
			return nil, blocked
		}
	}
	return targets, nil
}
func (s *AutomaticErasureService) erasePendingTargets(ctx context.Context, subject model.ErasureExternalSubject) error {
	targets, err := s.Store.ErasureTargets(subject.RequestID)
	if err != nil {
		return err
	}
	if len(targets) != len(model.ErasureTargetNames) {
		return &model.ErasureBlocked{Code: "ERASURE_TARGETS_REVIEW_REQUIRED"}
	}
	for _, target := range targets {
		if !target.Verified() {
			subject.RequiredTargets = append(subject.RequiredTargets, target.Name)
		}
	}
	if len(subject.RequiredTargets) == 0 {
		return s.Store.RecordExternalErasure(subject.RequestID, "verified-per-storage: see target evidence")
	}
	if err = s.Store.BeginErasureTargets(subject.RequestID); err != nil {
		return err
	}
	var results []model.ErasureTarget
	if processor, ok := s.External.(detailedErasureProcessor); ok {
		results, err = processor.EraseTargets(ctx, subject)
		if err == nil {
			results, err = validateExternalTargets(subject.RequiredTargets, results)
		}
	} else {
		var evidence string
		evidence, err = s.External.Erase(ctx, subject)
		if err == nil {
			for _, name := range subject.RequiredTargets {
				results = append(results, model.ErasureTarget{Name: name, Status: "complete", Evidence: evidence})
			}
			results, err = validateExternalTargets(subject.RequiredTargets, results)
		}
	}
	if err != nil {
		failed := []model.ErasureTarget{}
		for _, name := range subject.RequiredTargets {
			failed = append(failed, model.ErasureTarget{Name: name, Status: "failed"})
		}
		if saveErr := s.Store.RecordErasureTargets(subject.RequestID, failed); saveErr != nil {
			return saveErr
		}
		return err
	}
	if err = s.Store.RecordErasureTargets(subject.RequestID, results); err != nil {
		return err
	}
	for _, target := range results {
		if !target.Verified() {
			return &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
		}
	}
	return s.Store.RecordExternalErasure(subject.RequestID, "verified-per-storage: see target evidence")
}
