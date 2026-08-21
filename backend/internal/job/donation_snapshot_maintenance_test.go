// donation_snapshot_maintenance_test.go — Maintenance freeze tests for donation snapshots.
package job

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func TestDonationSnapshotRejectsManualWriteDuringMaintenance(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	job := &DonationSnapshotJob{maintenanceGate: gate, logger: zerolog.Nop()}

	if err := job.CreateSnapshotNow(); !errors.Is(err, maintenance.ErrWritesFrozen) {
		t.Fatalf("error = %v, want ErrWritesFrozen", err)
	}
}

func TestDonationSnapshotAdmittedContinuationCompletesAfterFreezeCloses(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	outerLease, err := gate.EnterWriter(false)
	if err != nil {
		t.Fatal(err)
	}
	defer outerLease.Release()
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT CAST(COALESCE(SUM(O_PRICE), 0) AS SIGNED)")).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1000))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT CAST(COUNT(DISTINCT USR_SEQ) AS SIGNED)")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("FROM DONATION_CONFIG").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO DONATION_SNAPSHOT").WillReturnResult(sqlmock.NewResult(1, 1))

	job := NewDonationSnapshotJob(repository.NewDonationRepository(sqlx.NewDb(rawDB, "sqlmock")), gate, zerolog.Nop())
	if err := job.CreateSnapshotNowAdmitted(); err != nil {
		t.Fatalf("admitted snapshot: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
