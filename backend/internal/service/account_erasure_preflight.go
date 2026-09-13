package service

import "github.com/dflh-saf/backend/internal/model"

// A manual handoff must be acknowledged before local identifiers are removed.
// "manual" is acceptance of work, never proof that the target has been erased.
func (s *AutomaticErasureService) externalReadyForDeletion(w model.ErasureWork) error {
	if s.External != nil || w.ExternalEvidence != "" {
		return nil
	}
	targets, err := s.Store.ErasureTargets(w.RequestID)
	if err != nil {
		return err
	}
	if len(targets) != len(model.ErasureTargetNames) {
		return &model.ErasureBlocked{Code: "ERASURE_TARGETS_REVIEW_REQUIRED"}
	}
	for _, target := range targets {
		if !target.Verified() && target.Status != "manual" {
			return &model.ErasureBlocked{Code: "EXTERNAL_HANDOFF_REVIEW_REQUIRED"}
		}
	}
	return nil
}
