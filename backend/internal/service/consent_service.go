// consent_service.go — Decides whether a signup request carries acceptable
// 개인정보 수집·이용 동의 and records it once the account exists.
package service

import (
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

var (
	// ErrConsentRequired means the request carried no consent while enforcement is on,
	// or the applicant explicitly declined.
	ErrConsentRequired = errors.New("privacy consent required")
	// ErrConsentVersionOutdated means the client showed a notice version other than the
	// one currently in force; the client must refresh its notice text.
	ErrConsentVersionOutdated = errors.New("privacy consent version outdated")
)

type consentStore interface {
	RecordAccepted(accountID int, consentType, version string, required bool, acceptedAt time.Time) error
}

// ConsentService applies PrivacyConsentConfig to signup requests. A nil store means
// AUTH_CONSENT is unavailable; evaluation still runs but nothing is persisted.
type ConsentService struct {
	store  consentStore
	cfg    config.PrivacyConsentConfig
	logger zerolog.Logger
	now    func() time.Time
}

// NewConsentService creates a ConsentService. Pass a nil store to evaluate without
// recording.
func NewConsentService(store consentStore, cfg config.PrivacyConsentConfig, logger zerolog.Logger) *ConsentService {
	return &ConsentService{store: store, cfg: cfg, logger: logger, now: time.Now}
}

// Evaluate decides whether a request may proceed. Rules:
//   - accepted=false is always rejected: the applicant declined.
//   - nil consent (client predates the consent UI) passes while Enforce is off and is
//     logged; it is rejected once Enforce is on.
//   - a version other than the configured one passes while Enforce is off (logged) and
//     is rejected once Enforce is on. An empty configured version disables the check.
func (s *ConsentService) Evaluate(consent *model.PrivacyConsent) error {
	if consent == nil {
		if s.cfg.Enforce {
			return ErrConsentRequired
		}
		s.logger.Warn().Msg("signup without privacy consent accepted (enforcement off; legacy client)")
		return nil
	}
	if !consent.Accepted {
		return ErrConsentRequired
	}
	if s.cfg.Version != "" && consent.Version != s.cfg.Version {
		if s.cfg.Enforce {
			return ErrConsentVersionOutdated
		}
		s.logger.Warn().Str("clientVersion", consent.Version).Str("currentVersion", s.cfg.Version).
			Msg("signup with outdated privacy consent version accepted (enforcement off)")
	}
	return nil
}

// Record persists an accepted consent for a newly created account. Nothing is written
// for nil consent (legacy client) or when the store is unavailable. Failures are
// returned for logging; the account already exists, so callers must not roll back.
func (s *ConsentService) Record(accountID int, consent *model.PrivacyConsent) error {
	if consent == nil || !consent.Accepted || s.store == nil {
		return nil
	}
	return s.store.RecordAccepted(accountID, model.ConsentTypePrivacy, consent.Version, true, s.now())
}
