package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestPushOutboxRepositoryEnqueueUsesDuplicateNoop(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewPushOutboxRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_PUSH_OUTBOX")).
		WithArgs(
			"message.new",
			"message.new:777:67890",
			67890,
			1,
			"token-1",
			"sandbox",
			nullStringArg{value: "com.daeil.dflhsafv2", valid: true},
			"새 쪽지가 도착했습니다",
			"새로운 쪽지가 도착했습니다.",
			`{"event_type":"message.new"}`,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Enqueue(context.Background(), PushOutboxInsert{
		EventType:       "message.new",
		EventID:         "message.new:777:67890",
		UsrSeq:          67890,
		MDTSeq:          1,
		DeviceToken:     "token-1",
		APNsEnvironment: "sandbox",
		BundleID:        "com.daeil.dflhsafv2",
		Title:           "새 쪽지가 도착했습니다",
		Body:            "새로운 쪽지가 도착했습니다.",
		PayloadJSON:     `{"event_type":"message.new"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPushOutboxRepositoryClaimDueUsesClaimToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	repo := NewPushOutboxRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ALUMNI_PUSH_OUTBOX")).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnResult(sqlmock.NewResult(0, 1))
	rows := sqlmock.NewRows([]string{
		"PO_SEQ", "EVENT_TYPE", "EVENT_ID", "USR_SEQ", "MDT_SEQ", "DEVICE_TOKEN", "APNS_ENVIRONMENT",
		"BUNDLE_ID", "TITLE", "BODY", "PAYLOAD_JSON", "STATUS", "ATTEMPT_COUNT", "NEXT_ATTEMPT_AT",
		"LAST_ERROR_CODE", "LAST_ERROR_MESSAGE", "CREATED_AT", "UPDATED_AT", "SENT_AT",
	}).AddRow(
		100, "message.new", "message.new:777:67890", 67890, 1, "token-1", "sandbox",
		"com.daeil.dflhsafv2", "title", "body", `{"event_type":"message.new"}`, "PROCESSING", 0, now,
		nil, nil, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT PO_SEQ, EVENT_TYPE, EVENT_ID")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	jobs, err := repo.ClaimDue(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].POSeq != 100 || jobs[0].Status != PushOutboxStatusProcessing {
		t.Fatalf("unexpected claimed jobs: %#v", jobs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPushOutboxRepositoryResetStuckProcessing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewPushOutboxRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta("STATUS = CASE WHEN LAST_ERROR_CODE = 'DELIVERY_STARTED' THEN 'DEAD' ELSE 'FAILED' END")).
		WithArgs(300).
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.ResetStuckProcessing(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 recovered jobs, got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPushOutboxRepositoryMarkDeliveryStarted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewPushOutboxRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta("LAST_ERROR_CODE = 'DELIVERY_STARTED'")).
		WithArgs(100).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkDeliveryStarted(context.Background(), 100); err != nil {
		t.Fatalf("MarkDeliveryStarted: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
