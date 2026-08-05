// maintenance_config_test.go — Runtime fail-closed maintenance defaults.
package config

import "testing"

func TestLoadDefaultsMaintenanceSentinelPath(t *testing.T) {
	t.Setenv("MAINTENANCE_SENTINEL_PATH", "")

	cfg := Load()

	if cfg.Maintenance.SentinelPath != "/run/alumni/maintenance" {
		t.Fatalf("sentinel path = %q, want canonical runtime path", cfg.Maintenance.SentinelPath)
	}
}
