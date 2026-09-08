package repository

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

// Used by the network=none runner: the compiled tests and MariaDB share only
// this disposable container's Unix socket. No configurable remote DSN exists.
func startIsolatedSocketMariaDB101(t *testing.T) *sqlx.DB {
	t.Helper()
	if _, err := os.Stat("/.dockerenv"); err != nil {
		t.Fatal("isolated socket tests require Docker")
	}
	password := os.Getenv("MYSQL_ROOT_PASSWORD")
	if password == "" {
		t.Fatal("missing disposable database password")
	}
	dsn := "root:" + password + "@unix(/var/run/mysqld/mysqld.sock)/"
	admin, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	var version string
	if err := admin.Get(&version, "SELECT VERSION()"); err != nil || !strings.HasPrefix(version, "10.1.38") {
		t.Fatal("unexpected test database version", version, err)
	}
	name := fmt.Sprintf("erasure_synthetic_%d", time.Now().UnixNano())
	admin.MustExec("CREATE DATABASE `" + name + "`")
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	})
	db, err := sqlx.Connect("mysql", dsn+name+"?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Restore synthetic snapshots before and after erasure. A pre-erasure backup
// intentionally restores the member, proving that live DB erasure alone is
// not sufficient evidence for marking the backup target complete.
func verifySyntheticErasureBackup(t *testing.T, source *sqlx.DB, expectedMemberCount int) {
	t.Helper()
	var sourceName string
	if err := source.Get(&sourceName, "SELECT DATABASE()"); err != nil {
		t.Fatal(err)
	}
	dump := exec.Command("mysqldump", "-uroot", "--socket=/var/run/mysqld/mysqld.sock", "--single-transaction", sourceName)
	dump.Env = append(os.Environ(), "MYSQL_PWD="+os.Getenv("MYSQL_ROOT_PASSWORD"))
	data, err := dump.Output()
	if err != nil {
		t.Fatal("dump synthetic backup", err)
	}
	restored := startIsolatedSocketMariaDB101(t)
	var restoredName string
	if err := restored.Get(&restoredName, "SELECT DATABASE()"); err != nil {
		t.Fatal(err)
	}
	restore := exec.Command("mysql", "-uroot", "--socket=/var/run/mysqld/mysqld.sock", restoredName)
	restore.Env = dump.Env
	restore.Stdin = bytes.NewReader(data)
	if output, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("restore synthetic backup: %v %s", err, output)
	}
	var count int
	if err := restored.Get(&count, "SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=42"); err != nil || count != expectedMemberCount {
		t.Fatal("restored member count", count, expectedMemberCount, err)
	}
	for _, query := range []string{
		"SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=43 AND USR_ID='fake43'",
		"SELECT COUNT(*) FROM WEO_BOARDBBS WHERE SEQ=11 AND CONTENTS='preserved other post'",
		"SELECT COUNT(*) FROM WEO_BOARDCOMAND WHERE SEQ=22 AND CONTENTS='preserved other reply'",
		"SELECT COUNT(*) FROM WEO_ORDER WHERE O_SEQ=2 AND O_NET_RECEIVED_AMOUNT=50",
	} {
		if err := restored.Get(&count, query); err != nil || count != 1 {
			t.Fatal("restore lost other member data", query, err)
		}
	}
	t.Logf("synthetic backup restored: target member rows=%d, unrelated member/content/donation preserved", expectedMemberCount)
}
