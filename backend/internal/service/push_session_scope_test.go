package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"time"
)

type guardedPushDeliveryStore struct {
	pushDeliveryStoreStub
	checks        int
	allowed       []bool
	checkErr      error
	scopedDeleted []model.PushDeliveryTarget
}

func (s *guardedPushDeliveryStore) DeliveryTargetStillCurrent(_ int, target model.PushDeliveryTarget) (bool, error) {
	s.checks++
	if s.checkErr != nil {
		return false, s.checkErr
	}
	if len(s.allowed) == 0 {
		return false, nil
	}
	allowed := s.allowed[0]
	s.allowed = s.allowed[1:]
	return allowed, nil
}
func (s *guardedPushDeliveryStore) DeleteDeliveryTarget(_ int, target model.PushDeliveryTarget) error {
	s.scopedDeleted = append(s.scopedDeleted, target)
	return nil
}

func TestPushDeliverySessionGuardSuppressesQueuedOrExpiredOwnership(t *testing.T) {
	for _, err := range []error{nil, errors.New("database unavailable")} {
		store := &guardedPushDeliveryStore{pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "ios", DeviceToken: "synthetic", SessionID: strings.Repeat("a", 32)}}}, checkErr: err}
		provider := &pushProviderStub{}
		notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
		notifier.deliver(context.Background(), pushDeliveryTestItem())
		if len(provider.calls) != 0 || store.checks != 1 {
			t.Fatalf("send=%d checks=%d", len(provider.calls), store.checks)
		}
	}
}
func TestPushDeliverySessionGuardRechecksRetryAndScopesInvalidCleanup(t *testing.T) {
	target := model.PushDeliveryTarget{Platform: "ios", DeviceToken: "synthetic", SessionID: strings.Repeat("a", 32)}
	for _, invalid := range []bool{false, true} {
		store := &guardedPushDeliveryStore{pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{target}}, allowed: []bool{true, false}}
		provider := &pushProviderStub{errors: []error{ErrPushTransient}}
		if invalid {
			provider.errors = []error{ErrPushInvalidToken}
		}
		notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
		notifier.deliver(context.Background(), pushDeliveryTestItem())
		if len(provider.calls) != 1 || len(store.deleted) != 0 {
			t.Fatalf("calls=%d unscoped deletes=%d", len(provider.calls), len(store.deleted))
		}
		if invalid {
			if len(store.scopedDeleted) != 1 || store.scopedDeleted[0] != target {
				t.Fatal("cleanup lost original ownership")
			}
		} else if store.checks != 2 {
			t.Fatal("retry failed to recheck")
		}
	}
}
func TestDeferredGlobalLogoutRejectsExpiredProofWithoutDatabase(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	now := time.Now()
	claims := jwt.MapClaims{"iss": mobileTokenIssuer, "aud": mobileTokenAudience, "sub": "42", "typ": "refresh", "ver": 1, "sid": strings.Repeat("a", 32), "jti": strings.Repeat("b", 32), "iat": now.Add(-2 * time.Hour).Unix(), "nbf": now.Add(-2 * time.Hour).Unix(), "exp": now.Add(-time.Hour).Unix()}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(auth.cfg.JWT.Secret))
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.RevokeAllSessionsWithOriginalProof(context.Background(), token); !errors.Is(err, repository.ErrRefreshTokenInvalid) {
		t.Fatalf("expired global error=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
