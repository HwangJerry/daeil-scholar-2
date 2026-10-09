package handler

import (
	"errors"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

type failingFinalizationStore struct{ *service.SocialLinkTokenStore }

func (s failingFinalizationStore) ConsumeWithPhoto(lease service.SocialLinkTokenLease, photo string) error {
	if err := s.SocialLinkTokenStore.ConsumeWithPhoto(lease, photo); err != nil {
		return err
	}
	return errors.New("synthetic postcommit finalization acknowledgement failure")
}

type finalizationPhoneStore struct {
	*repository.PhoneVerificationRepository
	ConsumedUser int
}

func (s *finalizationPhoneStore) ConsumeGrantForMember(_ string, phone string, user int) (string, error) {
	s.ConsumedUser = user
	return phone, nil
}

type finalizationConsentStore struct{ RecordedUser int }

func (s *finalizationConsentStore) RecordAccepted(user int, _, _ string, _ bool, _ time.Time) error {
	s.RecordedUser = user
	return nil
}

func TestCommittedSignupFinalizationFailureStillBindsGrantAndConsent(t *testing.T) {
	raw := cache.New(time.Minute, time.Minute)
	store := service.NewSocialLinkTokenStore(raw)
	_, _ = store.Put("terminal", model.SocialLinkData{AccessToken: "synthetic", Email: "synthetic@example.test"}, time.Minute)
	lease, err := store.Begin("terminal")
	if err != nil {
		t.Fatal(err)
	}
	phone := &finalizationPhoneStore{}
	consent := &finalizationConsentStore{}
	h := &AuthHandler{socialLinkTokens: failingFinalizationStore{store}, phoneVerifier: service.NewPhoneVerificationService(phone, nil, nil, zerolog.Nop()), consentSvc: service.NewConsentService(consent, config.PrivacyConsentConfig{}, zerolog.Nop()), logger: zerolog.Nop()}
	err = h.finalizeSocialSignup(lease, "", 42, true, socialLinkRequest{Phone: "01012345678", PhoneVerificationToken: "synthetic-grant", PrivacyConsent: &model.PrivacyConsent{Version: "synthetic", Accepted: true}})
	if err == nil {
		t.Fatal("fault injection did not fail")
	}
	if phone.ConsumedUser != 42 || consent.RecordedUser != 42 {
		t.Fatalf("committed proof/consent skipped: grant=%d consent=%d", phone.ConsumedUser, consent.RecordedUser)
	}
	if _, err := store.Begin("terminal"); !errors.Is(err, service.ErrSocialLinkTokenConsumed) {
		t.Fatalf("fault revived continuation: %v", err)
	}
	if _, err := store.Snapshot("terminal"); !errors.Is(err, service.ErrSocialLinkTokenConsumed) {
		t.Fatalf("fault exposes credentials: %v", err)
	}
}
