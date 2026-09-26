package mariadb

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestMain(m *testing.M) {
	os.Exit(Run(m))
}

func TestStartRequiresOptIn(t *testing.T) {
	t.Setenv("DFLH_DOCKER_TESTS", "")
	ran := false
	t.Run("disabled", func(t *testing.T) {
		Start(t)
		ran = true
	})
	if ran {
		t.Fatal("started MariaDB without opt-in")
	}
}

func TestMissingDockerIsUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := startServer(); !errors.Is(err, errDockerUnavailable) {
		t.Fatalf("missing Docker error = %v", err)
	}
}

func TestDatabasesAreIsolated(t *testing.T) {
	server := Start(t)
	schemaPath := filepath.Join(t.TempDir(), "schema.sql")
	if err := os.WriteFile(schemaPath, []byte("CREATE TABLE fixture (id INT PRIMARY KEY) ENGINE=InnoDB;"), 0o600); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	databases := make(map[string]*Database)
	t.Run("parallel", func(t *testing.T) {
		for _, name := range []string{"first", "second"} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				if Start(t) != server {
					t.Fatal("server was not shared")
				}
				database := server.NewDatabase(t, File(schemaPath), Statement("INSERT INTO fixture VALUES (42);"))
				config, err := mysql.ParseDSN(database.DSN)
				if err != nil {
					t.Fatal(err)
				}
				mu.Lock()
				_, duplicate := databases[config.DBName]
				databases[config.DBName] = database
				mu.Unlock()
				if duplicate || config.Addr != server.config.Addr {
					t.Fatal("expected distinct databases on the shared container")
				}
				connection, err := sqlx.Connect("mysql", database.DSN)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { connection.Close() })
				var id int
				if err := connection.Get(&id, "SELECT id FROM fixture"); err != nil || id != 42 {
					t.Fatalf("schema/seed not available through DSN: id=%d err=%v", id, err)
				}
			})
		}
	})
	if len(databases) != 2 {
		t.Fatalf("database count = %d, want 2", len(databases))
	}
	for name, database := range databases {
		if err := database.DB.Ping(); err == nil {
			t.Error("test connection pool was not closed")
		}
		var count int
		if err := server.admin.Get(&count, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name); err != nil || count != 0 {
			t.Fatalf("database survived test cleanup: count=%d err=%v", count, err)
		}
	}
}
