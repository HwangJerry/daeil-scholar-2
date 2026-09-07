// account_erasure_targets_test.go — Partial proof survives retries without repeating completed targets.
package service

import (
	"context"
	"github.com/dflh-saf/backend/internal/model"
	"testing"
)

type partialErasureFake struct {
	erasureExternalFake
	requests [][]string
}

func (f *partialErasureFake) EraseTargets(_ context.Context, s model.ErasureExternalSubject) ([]model.ErasureTarget, error) {
	f.requests = append(f.requests, s.RequiredTargets)
	results := []model.ErasureTarget{}
	for _, name := range s.RequiredTargets {
		target := model.ErasureTarget{Name: name, Status: "complete", Evidence: "synthetic-proof"}
		if len(f.requests) == 1 && name == "backups" {
			target.Status = "pending"
			target.Evidence = ""
		}
		if name == "external_data" {
			target.Status = "not_applicable"
		}
		results = append(results, target)
	}
	return results, nil
}
func TestPartialErasureResumesOnlyPendingStorage(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42}}
	external := &partialErasureFake{}
	s := &AutomaticErasureService{Store: store, External: external, ContextCipher: testContextCipher(t)}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 0 || store.databaseCalls != 1 || len(external.requests) != 1 {
		t.Fatal("partial proof became completion")
	}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 1 || store.databaseCalls != 1 || len(external.requests) != 2 || len(external.requests[1]) != 1 || external.requests[1][0] != "backups" {
		t.Fatal("verified storage was repeated", external.requests)
	}
}
func TestVerifiedTargetsRecoverAfterContextExpiry(t *testing.T) {
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42, Stage: "database_erased"}}
	for _, name := range model.ErasureTargetNames {
		store.targets = append(store.targets, model.ErasureTarget{Name: name, Status: "complete", Evidence: "synthetic-proof"})
	}
	s := &AutomaticErasureService{Store: store}
	if err := s.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 1 || store.databaseCalls != 0 {
		t.Fatal("durable proof ignored after restart")
	}
}
func TestExternalTargetEvidenceCannotBeInventedOrSubstituted(t *testing.T) {
	for _, targets := range [][]model.ErasureTarget{
		{{Name: "backups", Status: "not_applicable"}},
		{{Name: "other_identifiers", Status: "complete", Evidence: "proof"}},
		{{Name: "backups", Status: "complete", Evidence: "proof"}, {Name: "backups", Status: "complete", Evidence: "proof"}},
		{{Name: "backups", Status: "failed", Evidence: "private server response"}},
		{{Name: "backups", Status: "unknown", Evidence: "proof"}},
	} {
		if _, err := validateExternalTargets([]string{"backups"}, targets); err == nil {
			t.Fatal("invalid evidence accepted", targets)
		}
	}
	if _, err := validateExternalTargets([]string{"backups"}, []model.ErasureTarget{{Name: "backups", Status: "not_applicable", Evidence: "verified-inventory-reference"}}); err != nil {
		t.Fatal(err)
	}
}
