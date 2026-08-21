// visit_aggregation_maintenance_test.go — Maintenance freeze tests for visit aggregation.
package job

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

func TestVisitAggregationBackfillDoesNotTouchDatabaseDuringMaintenance(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()

	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	job := NewVisitAggregationJob(repository.NewVisitRepository(sqlx.NewDb(rawDB, "sqlmock")), gate, zerolog.Nop())
	job.backfillRecent()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestVisitScheduledCycleReleasesLeaseAfterPanic(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "maintenance")
	bridge := filepath.Join(dir, "maintenance-release-bridge")
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.ConfigureRelease(maintenance.ReleaseConfig{
		BridgePath:       bridge,
		ExpectedOwnerUID: os.Getuid(),
	}); err != nil {
		t.Fatal(err)
	}
	job := NewVisitAggregationJob(nil, gate, zerolog.Nop())
	func() {
		defer func() { _ = recover() }()
		job.runScheduledCycle(time.Now(), time.Now())
	}()
	generation := "0123456789abcdef0123456789abcdef"
	attempt := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := os.WriteFile(sentinel, []byte("state=active\ngeneration="+generation+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bridge, []byte("state=prepared\ngeneration="+generation+"\napproval_attempt_id="+attempt+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := gate.CloseAndDrain(ctx, generation, attempt); err != nil {
		t.Fatalf("panic leaked visit admission lease: %v", err)
	}
}
