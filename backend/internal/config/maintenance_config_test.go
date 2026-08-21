// maintenance_config_test.go — Runtime fail-closed maintenance defaults.
package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsMaintenanceSentinelPath(t *testing.T) {
	t.Setenv("MAINTENANCE_SENTINEL_PATH", "")
	t.Setenv("MAINTENANCE_RELEASE_BRIDGE_PATH", "")
	t.Setenv("MAINTENANCE_RELEASE_OWNER_UID", "")
	t.Setenv("MAINTENANCE_RELEASE_DRAIN_TIMEOUT", "")

	cfg := Load()

	if cfg.Maintenance.SentinelPath != "/run/alumni/maintenance" {
		t.Fatalf("sentinel path = %q, want canonical runtime path", cfg.Maintenance.SentinelPath)
	}
	if cfg.Maintenance.ReleaseBridgePath != "/run/alumni/maintenance-release-bridge" {
		t.Fatalf("release bridge path = %q, want canonical runtime path", cfg.Maintenance.ReleaseBridgePath)
	}
	if cfg.Maintenance.ReleaseOwnerUID != 0 {
		t.Fatalf("release owner UID = %d, want root", cfg.Maintenance.ReleaseOwnerUID)
	}
	if cfg.Maintenance.ReleaseDrainTimeout != 90*time.Second {
		t.Fatalf("release drain timeout = %s, want 90s", cfg.Maintenance.ReleaseDrainTimeout)
	}
}
