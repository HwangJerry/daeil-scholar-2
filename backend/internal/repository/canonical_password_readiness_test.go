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
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(6))
	mock.ExpectQuery(`_migration_history[\s\S]*043_finalize_identity_authority\.sql[\s\S]*044_enforce_account_lifecycle_invariants\.sql[\s\S]*information_schema\.TRIGGERS`).
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
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(6))
	mock.ExpectQuery(`EXISTS \(\s*SELECT 1 FROM _migration_history\s*WHERE filename = '043_finalize_identity_authority.sql'\s*\)[\s\S]*044_enforce_account_lifecycle_invariants.sql`).
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
	mock.ExpectQuery(`information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(6))
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

func TestPhoneClaimsWriteReadyDependsOnlyOnDurableMigration044(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`TABLE_NAME IN \('AUTH_PHONE_CLAIM', '_migration_history'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(2))
	mock.ExpectQuery(`FROM _migration_history`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))

	ready, err := PhoneClaimsWriteReady(sqlxDB)
	if err != nil || !ready {
		t.Fatalf("ready = %v, err = %v", ready, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPhoneClaimsWriteReadyReturnsFalseBeforeSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(`information_schema.TABLES`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))

	ready, err := PhoneClaimsWriteReady(sqlxDB)
	if err != nil || ready {
		t.Fatalf("ready = %v, err = %v", ready, err)
	}
}
