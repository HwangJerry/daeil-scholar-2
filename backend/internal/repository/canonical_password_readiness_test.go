package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestCanonicalPasswordWriteReadyReturnsFalseBeforeCanonicalSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))

	ready, err := CanonicalPasswordWriteReady(sqlxDB)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("CanonicalPasswordWriteReady() = true before schema")
	}
}

func TestCanonicalPasswordWriteReadyRequiresAuthorityFinalization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(5))
	mock.ExpectQuery(`_migration_history[\s\S]*043_finalize_identity_authority\.sql[\s\S]*information_schema\.TRIGGERS`).
		WillReturnRows(sqlmock.NewRows([]string{"READY"}).AddRow(0))

	ready, err := CanonicalPasswordWriteReady(sqlxDB)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("CanonicalPasswordWriteReady() = true before authority finalization")
	}
}

func TestCanonicalPasswordWriteReadyRequiresAppliedConflictFreeBackfill(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(5))
	mock.ExpectQuery(`EXISTS \(\s*SELECT 1 FROM _migration_history\s*WHERE filename = '043_finalize_identity_authority.sql'\s*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"READY"}).AddRow(1))

	ready, err := CanonicalPasswordWriteReady(sqlxDB)
	if err != nil {
		t.Fatal(err)
	}
	if !ready {
		t.Fatal("CanonicalPasswordWriteReady() = false for applied backfill")
	}
}

func TestCanonicalPasswordWriteReadyChecksLatestRunRatherThanLatestPassingRun(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(5))
	mock.ExpectQuery(`WHERE run.RUN_ID = \(\s*SELECT latest.RUN_ID\s*FROM AUTH_IDENTITY_MIGRATION_RUN latest\s*ORDER BY latest.STARTED_AT DESC, latest.RUN_ID DESC\s*LIMIT 1\s*\)`).
		WillReturnRows(sqlmock.NewRows([]string{"READY"}).AddRow(0))

	ready, err := CanonicalPasswordWriteReady(sqlxDB)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("CanonicalPasswordWriteReady() = true when the latest run is not ready")
	}
}
