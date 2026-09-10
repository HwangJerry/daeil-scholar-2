package service

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

type reviewNotifierStub struct {
	users    []int
	statuses []model.VerificationStatus
}

func (n *reviewNotifierStub) NotifyVerificationReviewed(user int, status model.VerificationStatus) {
	n.users = append(n.users, user)
	n.statuses = append(n.statuses, status)
}

func TestReviewNotifiesOnlyAfterSuccessfulStatusWrite(t *testing.T) {
	for _, approved := range []bool{true, false} {
		for _, fails := range []bool{true, false} {
			db, mock, _ := sqlmock.New()
			service := NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock")))
			notifier := &reviewNotifierStub{}
			service.SetVerificationReviewNotifier(notifier)
			update := mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`)
			if fails {
				update.WillReturnError(errors.New("write failed"))
			} else {
				update.WillReturnResult(sqlmock.NewResult(0, 1))
			}
			var err error
			status := model.VerificationRejected
			if approved {
				status = model.VerificationApproved
				err = service.ApproveAlumniVerification(42, 7, time.Now())
			} else {
				err = service.RejectAlumniVerification(42, 7, "사유", time.Now())
			}
			if fails {
				if err == nil || len(notifier.users) != 0 {
					t.Fatal("failed review notified")
				}
			} else {
				if err != nil || len(notifier.users) != 1 || notifier.users[0] != 42 || notifier.statuses[0] != status {
					t.Fatal("review notification missing or wrong recipient")
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			db.Close()
		}
	}
}

type reviewDeliveryStore struct {
	pushDeliveryStoreStub
	current bool
}

func (s *reviewDeliveryStore) VerificationStillCurrent(int, model.VerificationStatus) (bool, error) {
	return s.current, nil
}

func TestReviewDeliveryTargetsBothPlatformsAndIgnoresMessagePreference(t *testing.T) {
	for _, status := range []model.VerificationStatus{model.VerificationApproved, model.VerificationRejected} {
		store := &reviewDeliveryStore{pushDeliveryStoreStub: pushDeliveryStoreStub{preferences: &model.PushPreferences{MessageEnabled: false}, targets: []model.PushDeliveryTarget{{Platform: "ios", DeviceToken: "ios"}, {Platform: "android", DeviceToken: "android"}}}, current: true}
		provider := &pushProviderStub{}
		notifier := NewPushDeliveryNotifier(store, provider, zerolog.Nop())
		notifier.NotifyVerificationReviewed(42, status)
		item := <-notifier.shards[42%pushShardCount]
		notifier.deliver(context.Background(), item)
		if len(provider.calls) != 2 || provider.payloads[0].RecipientUserSeq != "42" || provider.payloads[0].Type != "verification.reviewed" {
			t.Fatal("review did not reach both platforms")
		}
		store.current = false
		notifier.deliver(context.Background(), item)
		if len(provider.calls) != 2 {
			t.Fatal("obsolete review or deleted account notified")
		}
	}
}
