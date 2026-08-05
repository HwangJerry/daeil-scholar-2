// donation_snapshot_maintenance_test.go — Maintenance freeze tests for donation snapshots.
package job

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dflh-saf/backend/internal/maintenance"
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
