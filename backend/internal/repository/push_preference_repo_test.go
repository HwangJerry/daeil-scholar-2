package repository

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestPushPreferenceRepositoryGetPreferencesReturnsDefaultsWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewPushPreferenceRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT NOTICE_ENABLED, MESSAGE_ENABLED")).
		WithArgs(10).
		WillReturnError(sql.ErrNoRows)

	preferences, err := repo.GetPreferences(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !preferences.NoticeEnabled || !preferences.MessageEnabled {
		t.Fatalf("expected default preferences enabled, got %#v", preferences)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPushPreferenceRepositoryUpsertPreferencesStoresFlags(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewPushPreferenceRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO ALUMNI_PUSH_PREFERENCE")).
		WithArgs(10, "N", "Y").
		WillReturnResult(sqlmock.NewResult(1, 1))

	preferences, err := repo.UpsertPreferences(10, model.PushPreferences{
		NoticeEnabled:  false,
		MessageEnabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preferences.NoticeEnabled || !preferences.MessageEnabled {
		t.Fatalf("unexpected preferences: %#v", preferences)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
