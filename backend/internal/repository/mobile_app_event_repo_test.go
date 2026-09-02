package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestMobileAppEventRepositoryInsertsBatchInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := NewMobileAppEventRepository(sqlx.NewDb(db, "sqlmock"))

	occurredAt := time.Date(2026, 9, 2, 12, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	createdAt := occurredAt.Add(time.Minute)
	userID := 42
	deviceModel := "iPhone17,1"
	events := []model.MobileAppEvent{
		{
			Platform: model.MobilePlatformIOS, EventType: model.MobileEventSignupStart,
			AppVersion: "2.3.0", OSVersion: "18.6", OccurredAt: occurredAt, CreatedAt: createdAt,
		},
		{
			Platform: model.MobilePlatformIOS, EventType: model.MobileEventSignupComplete,
			UserID: &userID, AppVersion: "2.3.0", OSVersion: "18.6", DeviceModel: &deviceModel,
			OccurredAt: occurredAt.Add(time.Second), CreatedAt: createdAt,
		},
	}

	mock.ExpectBegin()
	prepared := mock.ExpectPrepare(`INSERT INTO ALUMNI_MOBILE_APP_EVENT`)
	prepared.ExpectExec().
		WithArgs("ios", "signup_start", nil, "2.3.0", "18.6", nil, occurredAt, createdAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	prepared.ExpectExec().
		WithArgs("ios", "signup_complete", userID, "2.3.0", "18.6", deviceModel, occurredAt.Add(time.Second), createdAt).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	if err := repository.InsertBatch(events); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMobileAppEventRepositoryRollsBackFailedBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := NewMobileAppEventRepository(sqlx.NewDb(db, "sqlmock"))
	event := model.MobileAppEvent{
		Platform: model.MobilePlatformAndroid, EventType: model.MobileEventApplyComplete,
		AppVersion: "3.1.0", OSVersion: "16", OccurredAt: time.Now(), CreatedAt: time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO ALUMNI_MOBILE_APP_EVENT`).
		ExpectExec().
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	if err := repository.InsertBatch([]model.MobileAppEvent{event}); err == nil {
		t.Fatal("InsertBatch returned nil error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMobileAppEventRepositoryGetsGroupedSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := NewMobileAppEventRepository(sqlx.NewDb(db, "sqlmock"))
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	toExclusive := from.AddDate(0, 1, 0)

	mock.ExpectQuery(`SELECT PLATFORM, EVENT_TYPE, COUNT\(\*\) AS EVENT_COUNT`).
		WithArgs(from, toExclusive, "ios", "signup_start").
		WillReturnRows(sqlmock.NewRows([]string{"PLATFORM", "EVENT_TYPE", "EVENT_COUNT"}).
			AddRow("android", "signup_start", 11).
			AddRow("ios", "signup_start", 15))

	items, err := repository.GetSummary(model.MobileAppEventSummaryFilter{
		From: from, ToExclusive: toExclusive, Platform: "ios", EventType: "signup_start",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Count != 11 || items[1].Platform != "ios" {
		t.Fatalf("items = %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
