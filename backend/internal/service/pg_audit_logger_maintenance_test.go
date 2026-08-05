// pg_audit_logger_maintenance_test.go — Maintenance freeze behavior for PG audit storage.
package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPGAuditLoggerDoesNotCreateFilesWhileMaintenanceIsActive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "pg-audit.log")
	logger, err := NewPGAuditLogger(path, true)
	if err != nil {
		t.Fatalf("NewPGAuditLogger: %v", err)
	}
	logger.Log("fixture-order", "fixture-event", nil, nil)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("maintenance audit directory was created: %v", err)
	}
}
