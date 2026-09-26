package mariadb

import (
	"path/filepath"
	"testing"
)

const (
	prodBaselineSchemaFile     = "prod_baseline_schema_20260927.sql"
	prodBaselineMigrationsFile = "prod_baseline_applied_migrations_20260927.sql"
)

// ProdBaseline returns the production schema (no rows) as of migration 077 plus
// its _migration_history rows, so a fresh database matches production. Newer
// migrations can be applied on top with File.
func ProdBaseline(t testing.TB) []SQL {
	t.Helper()
	root, err := backendRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "migrations", "testdata")
	return []SQL{
		File(filepath.Join(dir, prodBaselineSchemaFile)),
		File(filepath.Join(dir, prodBaselineMigrationsFile)),
	}
}
