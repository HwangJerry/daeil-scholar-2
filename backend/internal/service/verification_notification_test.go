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
		notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(nil), zerolog.Nop())
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

// TestReviewDeliveryRendersTheDefaultReviewText pins the approved and rejected
// wording while nothing has been edited. The body still carries no rejection
// detail: those are read in the authenticated app, never on a lock screen.
func TestReviewDeliveryRendersTheDefaultReviewText(t *testing.T) {
	for _, test := range []struct {
		status   model.VerificationStatus
		wantBody string
		wantKey  string
	}{
		{model.VerificationApproved, "동문 인증이 승인되었습니다. 이제 동문 커뮤니티를 이용할 수 있어요.", model.NotificationTemplateVerificationApproved},
		{model.VerificationRejected, "동문 인증 신청이 반려되었습니다. 앱에서 사유를 확인하고 다시 신청해 주세요.", model.NotificationTemplateVerificationRejected},
	} {
		payload := deliverReview(t, test.status, nil)
		if payload.Title != "동문 인증 결과" || payload.SenderName != "동문 인증 결과" ||
			payload.Body != test.wantBody || payload.Preview != test.wantBody ||
			payload.TemplateKey != test.wantKey || payload.VerificationStatus != test.status {
			t.Fatalf("payload = %#v", payload)
		}
	}
}

// TestReviewDeliveryKeepsTheSenderNameWhenTheTitleIsEdited proves the routing
// label survives an administrator editing the displayed title.
func TestReviewDeliveryKeepsTheSenderNameWhenTheTitleIsEdited(t *testing.T) {
	payload := deliverReview(t, model.VerificationApproved, map[string]model.NotificationTemplate{
		model.NotificationTemplateVerificationApproved: {
			Key:     model.NotificationTemplateVerificationApproved,
			Channel: model.NotificationChannelPush,
			Title:   "인증이 끝났어요",
			Body:    "이제 동문 커뮤니티를 이용할 수 있어요.",
			Version: 2,
		},
	})
	if payload.Title != "인증이 끝났어요" || payload.Body != "이제 동문 커뮤니티를 이용할 수 있어요." {
		t.Fatalf("displayed text did not follow the edit: %#v", payload)
	}
	if payload.SenderName != verificationReviewSenderName {
		t.Fatalf("senderName = %q, want the fixed review label", payload.SenderName)
	}
}

// deliverReview runs one review notification end to end through the queue and
// returns the payload the provider received.
func deliverReview(
	t *testing.T,
	status model.VerificationStatus,
	rows map[string]model.NotificationTemplate,
) model.PushMessagePayload {
	t.Helper()
	store := &reviewDeliveryStore{
		pushDeliveryStoreStub: pushDeliveryStoreStub{targets: []model.PushDeliveryTarget{{Platform: "android", DeviceToken: "android"}}},
		current:               true,
	}
	provider := &pushProviderStub{}
	notifier := NewPushDeliveryNotifier(store, provider, NewTestNotificationTemplateService(rows), zerolog.Nop())
	notifier.NotifyVerificationReviewed(42, status)
	notifier.deliver(context.Background(), <-notifier.shards[42%pushShardCount])
	if len(provider.payloads) != 1 {
		t.Fatalf("provider payloads = %d, want 1", len(provider.payloads))
	}
	return provider.payloads[0]
}
