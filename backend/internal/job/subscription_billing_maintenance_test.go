// subscription_billing_maintenance_test.go — Maintenance freeze tests for recurring billing.
package job

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/rs/zerolog"
)

func TestSubscriptionBillingRejectsRunDuringMaintenance(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	job := &SubscriptionBillingJob{maintenanceGate: gate, logger: zerolog.Nop()}

	processed, runErrs := job.RunOnce(time.Now())
	if processed != 0 || len(runErrs) != 1 || !errors.Is(runErrs[0], maintenance.ErrWritesFrozen) {
		t.Fatalf("processed = %d errors = %v, want maintenance rejection", processed, runErrs)
	}
}
