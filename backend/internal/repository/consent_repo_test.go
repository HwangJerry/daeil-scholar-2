package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestConsentRecordAcceptedUpsertsOnAccountTypeVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := &ConsentRepository{DB: sqlx.NewDb(db, "sqlmock")}
	at := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)

	mock.ExpectExec(`INSERT INTO AUTH_CONSENT[\s\S]*VALUES \(\?, \?, \?, \?, 1, \?, NULL, \?\)[\s\S]*ON DUPLICATE KEY UPDATE IS_ACCEPTED = 1, ACCEPTED_AT = VALUES\(ACCEPTED_AT\), WITHDRAWN_AT = NULL`).
		WithArgs(42, "PRIVACY", "2026-09-30", true, at, at).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.RecordAccepted(42, "PRIVACY", "2026-09-30", true, at); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
