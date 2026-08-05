// session_cleanup_test.go — Maintenance freeze tests for session cleanup writes.
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

func TestSessionCleanupJobDoesNotTouchDatabaseDuringMaintenance(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	db := sqlx.NewDb(rawDB, "sqlmock")

	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := NewSessionCleanupJob(
		repository.NewSessionRepository(db),
		repository.NewPasswordResetRepository(db),
		repository.NewAuthRepository(db),
		gate,
		zerolog.Nop(),
	)
	cleanup.RunOnce()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
