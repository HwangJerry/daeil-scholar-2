package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

type recordingConsentStore struct {
	accountID   int
	consentType string
	version     string
	required    bool
	calls       int
}

func (r *recordingConsentStore) RecordAccepted(accountID int, consentType, version string, required bool, _ time.Time) error {
	r.calls++
	r.accountID, r.consentType, r.version, r.required = accountID, consentType, version, required
	return nil
}

func accepted(version string) *model.PrivacyConsent {
	return &model.PrivacyConsent{Version: version, Accepted: true}
}

func TestConsentEvaluateWhileEnforcementOff(t *testing.T) {
	svc := service.NewConsentService(nil, config.PrivacyConsentConfig{Version: "2026-09-30", Enforce: false}, zerolog.Nop())

	if err := svc.Evaluate(nil); err != nil {
		t.Fatalf("legacy client without consent must pass during rollout, got %v", err)
	}
	if err := svc.Evaluate(accepted("2026-09-01")); err != nil {
		t.Fatalf("outdated version must pass during rollout, got %v", err)
	}
	if err := svc.Evaluate(accepted("2026-09-30")); err != nil {
		t.Fatalf("current version must pass, got %v", err)
	}
	if err := svc.Evaluate(&model.PrivacyConsent{Version: "2026-09-30", Accepted: false}); !errors.Is(err, service.ErrConsentRequired) {
		t.Fatalf("explicit refusal must be rejected even during rollout, got %v", err)
	}
}

func TestConsentEvaluateWhileEnforcementOn(t *testing.T) {
	svc := service.NewConsentService(nil, config.PrivacyConsentConfig{Version: "2026-09-30", Enforce: true}, zerolog.Nop())

	if err := svc.Evaluate(nil); !errors.Is(err, service.ErrConsentRequired) {
		t.Fatalf("missing consent must be rejected, got %v", err)
	}
	if err := svc.Evaluate(accepted("2026-09-01")); !errors.Is(err, service.ErrConsentVersionOutdated) {
		t.Fatalf("outdated version must be rejected, got %v", err)
	}
	if err := svc.Evaluate(accepted("2026-09-30")); err != nil {
		t.Fatalf("current version must pass, got %v", err)
	}
}

func TestConsentEvaluateSkipsVersionCheckWhenUnconfigured(t *testing.T) {
	svc := service.NewConsentService(nil, config.PrivacyConsentConfig{Version: "", Enforce: true}, zerolog.Nop())
	if err := svc.Evaluate(accepted("anything")); err != nil {
		t.Fatalf("empty configured version disables the version check, got %v", err)
	}
}

func TestConsentRecordWritesOnlyAcceptedConsent(t *testing.T) {
	store := &recordingConsentStore{}
	svc := service.NewConsentService(store, config.PrivacyConsentConfig{Version: "2026-09-30"}, zerolog.Nop())

	if err := svc.Record(42, nil); err != nil || store.calls != 0 {
		t.Fatalf("nil consent must not be recorded: err=%v calls=%d", err, store.calls)
	}
	if err := svc.Record(42, accepted("2026-09-30")); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if store.calls != 1 || store.accountID != 42 || store.consentType != model.ConsentTypePrivacy || store.version != "2026-09-30" || !store.required {
		t.Fatalf("unexpected record: %+v", store)
	}
}

func TestConsentRecordWithoutStoreIsNoop(t *testing.T) {
	svc := service.NewConsentService(nil, config.PrivacyConsentConfig{}, zerolog.Nop())
	if err := svc.Record(42, accepted("2026-09-30")); err != nil {
		t.Fatalf("missing table must not fail signup, got %v", err)
	}
}
