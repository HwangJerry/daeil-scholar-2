// Package mariadb provides a disposable MariaDB 10.1.38 server for package tests.
package mariadb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

const queryTimeout = 15 * time.Second

var (
	managed   bool
	startOnce sync.Once
	shared    *Server
	startErr  error
)

// Run belongs in TestMain: os.Exit(mariadb.Run(m)). The server starts lazily and
// stays alive until all tests (including parallel tests and cleanups) finish.
func Run(m *testing.M) (code int) {
	managed = true
	defer func() {
		if shared != nil {
			if err := shared.close(); err != nil {
				fmt.Fprintln(os.Stderr, "clean up test MariaDB:", err)
				code = 1
			}
		}
	}()
	return m.Run()
}

// Start shares one server per test binary. DFLH_DOCKER_TESTS=1 enables it;
// additional opt-in variable names can preserve a caller's legacy test gate.
// A missing Docker CLI or unavailable daemon skips the calling test.
func Start(t testing.TB, optInEnv ...string) *Server {
	t.Helper()
	enabled := os.Getenv("DFLH_DOCKER_TESTS") == "1"
	for _, name := range optInEnv {
		enabled = enabled || os.Getenv(name) == "1"
	}
	if !enabled {
		t.Skip("set DFLH_DOCKER_TESTS=1 to run pinned MariaDB integration")
	}
	if !managed {
		t.Fatal("MariaDB tests require TestMain calling os.Exit(mariadb.Run(m))")
	}
	startOnce.Do(func() { shared, startErr = startServer() })
	if errors.Is(startErr, errDockerUnavailable) {
		t.Skipf("Docker unavailable: %v", startErr)
	}
	if startErr != nil {
		t.Fatalf("start pinned MariaDB: %v", startErr)
	}
	return shared
}

// Database owns a fresh database and its connection pool until t.Cleanup.
type Database struct {
	DB  *sqlx.DB
	DSN string
}

// SQL is one schema/fixture input. Inputs execute in the supplied order.
type SQL struct {
	file      string
	statement string
}

// File reads SQL relative to the test's working directory, without splitting it.
// Inputs must be server SQL, not mysql-client commands such as DELIMITER.
func File(path string) SQL { return SQL{file: path} }

// Statement accepts one or more SQL statements, including synthetic seed data.
func Statement(sql string) SQL { return SQL{statement: sql} }

func (s *Server) NewDatabase(t testing.TB, inputs ...SQL) *Database {
	t.Helper()
	name, err := randomName("t_")
	if err != nil {
		t.Fatal(err)
	}
	if err := execute(s.admin, "CREATE DATABASE `"+name+"` DEFAULT CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := execute(s.admin, "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	config := s.config.Clone()
	config.DBName = name
	dsn := config.FormatDSN()
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	for index, input := range inputs {
		statement := input.statement
		if input.file != "" {
			data, err := os.ReadFile(input.file)
			if err != nil {
				t.Fatalf("read SQL file %s: %v", input.file, err)
			}
			statement = string(data)
		}
		if strings.TrimSpace(statement) == "" {
			t.Fatalf("SQL input %d is empty", index+1)
		}
		if err := execute(db, statement); err != nil {
			t.Fatalf("apply SQL input %d (%s): %v", index+1, input.file, err)
		}
	}
	return &Database{DB: db, DSN: dsn}
}

func execute(db *sqlx.DB, statement string) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	_, err := db.ExecContext(ctx, statement)
	return err
}

func randomName(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes[:]), nil
}
