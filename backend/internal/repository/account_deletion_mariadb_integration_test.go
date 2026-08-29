package repository

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestPhoneClaimWriteLockSerializesMigrationStartOnMariaDB101(t *testing.T) {
	if os.Getenv("PASSWORD_RESET_DOCKER_INTEGRATION") != "1" {
		t.Skip("set PASSWORD_RESET_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}
	db := startPasswordResetMariaDB101(t)
	if _, err := db.Exec(`
		CREATE TABLE _migration_journal (
			filename VARCHAR(255) NOT NULL PRIMARY KEY,
			sha256 CHAR(64) NOT NULL,
			state VARCHAR(20) NOT NULL,
			started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME NULL
		) ENGINE=InnoDB
	`); err != nil {
		t.Fatal(err)
	}

	writeTx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	ready, err := detectPhoneClaimsInWriteTransaction(writeTx)
	if err != nil || ready {
		t.Fatalf("ready = %v, err = %v", ready, err)
	}

	migrationStarted := make(chan error, 1)
	go func() {
		_, insertErr := db.Exec(`
			INSERT INTO _migration_journal (filename, sha256, state)
			VALUES ('044_enforce_account_lifecycle_invariants.sql', REPEAT('a', 64), 'STARTED')
		`)
		migrationStarted <- insertErr
	}()
	select {
	case err := <-migrationStarted:
		t.Fatalf("migration STARTED insert bypassed write gap lock: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
	if err := writeTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-migrationStarted; err != nil {
		t.Fatal(err)
	}

	blockedTx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer blockedTx.Rollback()
	_, err = detectPhoneClaimsInWriteTransaction(blockedTx)
	if !errors.Is(err, ErrPhoneClaimsMigrating) {
		t.Fatalf("error = %v, want ErrPhoneClaimsMigrating", err)
	}
}

func TestAccountDeletionUpdatesOnlyMemberStatusOnMariaDB101(t *testing.T) {
	if os.Getenv("PASSWORD_RESET_DOCKER_INTEGRATION") != "1" {
		t.Skip("set PASSWORD_RESET_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	db := startPasswordResetMariaDB101(t)
	createAccountDeletionIntegrationSchema(t, db)
	repo := NewAuthRepository(db)

	if err := repo.AnonymizeAccountForDeletion(42); err != nil {
		t.Fatal(err)
	}

	var member struct {
		Status string `db:"USR_STATUS"`
		Name   string `db:"USR_NAME"`
		Phone  string `db:"USR_PHONE"`
		Email  string `db:"USR_EMAIL"`
		Nick   string `db:"USR_NICK"`
		Photo  string `db:"USR_PHOTO"`
	}
	if err := db.Get(&member, `
		SELECT USR_STATUS, USR_NAME, USR_PHONE, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_SEQ = 42
	`); err != nil {
		t.Fatal(err)
	}
	if member.Status != "AAA" {
		t.Fatalf("USR_STATUS = %q, want AAA", member.Status)
	}
	if member.Name != "홍길동" || member.Phone != "010-1234-5678" ||
		member.Email != "member@example.com" || member.Nick != "길동" || member.Photo != "profile.jpg" {
		t.Fatalf("account deletion changed PII: %#v", member)
	}
}

func createAccountDeletionIntegrationSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE WEO_MEMBER (
			USR_SEQ INT PRIMARY KEY,
			USR_STATUS CHAR(3) NOT NULL,
			USR_NAME VARCHAR(100) NOT NULL,
			USR_PHONE VARCHAR(32) NOT NULL,
			USR_EMAIL VARCHAR(255) NOT NULL,
			USR_NICK VARCHAR(100) NOT NULL,
			USR_PHOTO VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

		INSERT INTO WEO_MEMBER
			(USR_SEQ, USR_STATUS, USR_NAME, USR_PHONE, USR_EMAIL, USR_NICK, USR_PHOTO)
		VALUES
			(42, 'CCC', '홍길동', '010-1234-5678', 'member@example.com', '길동', 'profile.jpg')
	`)
	if err != nil {
		t.Fatal(err)
	}
}
