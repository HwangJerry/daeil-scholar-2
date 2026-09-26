package mariadb

import "testing"

const (
	prodBaselineTableCount   = 103
	prodBaselineTriggerCount = 7
	prodBaselineMigrations   = 77
)

func TestProdBaselineMatchesProductionShape(t *testing.T) {
	db := Start(t).NewDatabase(t, ProdBaseline(t)...).DB

	var tables, triggers, migrations int
	if err := db.Get(&tables, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE()`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&triggers, `SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = DATABASE()`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&migrations, `SELECT COUNT(*) FROM _migration_history`); err != nil {
		t.Fatal(err)
	}
	if tables != prodBaselineTableCount || triggers != prodBaselineTriggerCount || migrations != prodBaselineMigrations {
		t.Fatalf("baseline shape = %d tables, %d triggers, %d migrations; want %d, %d, %d",
			tables, triggers, migrations, prodBaselineTableCount, prodBaselineTriggerCount, prodBaselineMigrations)
	}

	// The legacy tables the numbered migrations assume must exist.
	for _, table := range []string{"WEO_MEMBER", "WEO_BOARDBBS", "FUNDAMENTAL_MEMBER", "AUTH_PHONE_CLAIM"} {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, table); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("baseline is missing table %s", table)
		}
	}
}
