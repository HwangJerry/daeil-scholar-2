// visit_aggregation_maintenance_test.go — Maintenance freeze tests for visit aggregation.
package job

import (
	"os"
	"path/filepath"
	"testing"

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
