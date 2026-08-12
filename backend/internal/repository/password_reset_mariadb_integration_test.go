package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestPasswordResetRepositorySerializesDistinctTokensOnMariaDB101(t *testing.T) {
	if os.Getenv("PASSWORD_RESET_DOCKER_INTEGRATION") != "1" {
		t.Skip("set PASSWORD_RESET_DOCKER_INTEGRATION=1 to run pinned MariaDB integration")
	}

	db := startPasswordResetMariaDB101(t)
	repo := NewPasswordResetRepository(db)
	createPasswordResetIntegrationSchema(t, db)

	type attempt struct {
		token      string
		legacyHash string
		canonical  string
	}
	attempts := []attempt{
		{token: "first-reset-token", legacyHash: "legacy-first", canonical: "canonical-first"},
		{token: "second-reset-token", legacyHash: "legacy-second", canonical: "canonical-second"},
	}
	start := make(chan struct{})
	errorsByAttempt := make([]error, len(attempts))
	var waitGroup sync.WaitGroup
	for index, candidate := range attempts {
		waitGroup.Add(1)
		go func(index int, candidate attempt) {
			defer waitGroup.Done()
			<-start
			errorsByAttempt[index] = repo.ConfirmResetAtomically(
				candidate.token+"-hash",
				candidate.token,
				candidate.legacyHash,
				passwordResetIntegrationCredential(candidate.canonical),
			)
		}(index, candidate)
	}
	close(start)
	waitGroup.Wait()

	winner := -1
	for index, err := range errorsByAttempt {
		if err == nil {
			if winner != -1 {
				t.Fatalf("multiple distinct reset tokens committed: %#v", errorsByAttempt)
			}
			winner = index
			continue
		}
		if !errors.Is(err, ErrPasswordResetTokenInvalid) {
			t.Fatalf("confirmation %d error = %v", index, err)
		}
	}
	if winner == -1 {
		t.Fatalf("no reset token committed: %#v", errorsByAttempt)
	}

	var unusedTokenCount int
	if err := db.Get(&unusedTokenCount, `SELECT COUNT(*) FROM ALUMNI_PASSWORD_RESET WHERE APR_USED_YN = 'N'`); err != nil {
		t.Fatal(err)
	}
	if unusedTokenCount != 0 {
		t.Fatalf("unused reset tokens = %d, want 0", unusedTokenCount)
	}
	var stored struct {
		Legacy    string `db:"USR_PWD"`
		Canonical string `db:"PASSWORD_HASH"`
	}
	if err := db.Get(&stored, `
		SELECT member.USR_PWD, credential.PASSWORD_HASH
		FROM WEO_MEMBER member
		JOIN AUTH_IDENTITY identity ON identity.ACCOUNT_ID = member.USR_SEQ
		JOIN AUTH_PASSWORD_CREDENTIAL credential ON credential.IDENTITY_ID = identity.IDENTITY_ID
		WHERE member.USR_SEQ = 42
	`); err != nil {
		t.Fatal(err)
	}
	if stored.Legacy != attempts[winner].legacyHash || stored.Canonical != attempts[winner].canonical {
		t.Fatalf("winning reset was overwritten: legacy=%q canonical=%q winner=%d", stored.Legacy, stored.Canonical, winner)
	}
}

func startPasswordResetMariaDB101(t *testing.T) *sqlx.DB {
	t.Helper()
	imageBytes, err := os.ReadFile("../../migrations/testdata/mariadb-10.1.38.image")
	if err != nil {
		t.Fatal(err)
	}
	passwordBytes := make([]byte, 24)
	if _, err := rand.Read(passwordBytes); err != nil {
		t.Fatal(err)
	}
	password := base64.RawURLEncoding.EncodeToString(passwordBytes)
	containerName := fmt.Sprintf("password-reset-integration-%d", time.Now().UnixNano())
	command := exec.Command("docker", "run", "-d", "--name", containerName,
		"-e", "MYSQL_ROOT_PASSWORD="+password, "-e", "MYSQL_DATABASE=password_reset_test",
		"-p", "127.0.0.1::3306", strings.TrimSpace(string(imageBytes)))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("start pinned MariaDB: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
	})
	portOutput, err := exec.Command("docker", "port", containerName, "3306/tcp").Output()
	if err != nil {
		t.Fatal(err)
	}
	address := strings.TrimSpace(string(portOutput))
	dsn := fmt.Sprintf("root:%s@tcp(%s)/password_reset_test?parseTime=true&multiStatements=true", password, address)
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := sqlx.NewDb(database, "mysql")
	t.Cleanup(func() { _ = db.Close() })
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return db
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("pinned MariaDB did not become ready")
	return nil
}

func createPasswordResetIntegrationSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY, USR_PWD VARCHAR(255) NOT NULL) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_PASSWORD_RESET (
			APR_SEQ BIGINT AUTO_INCREMENT PRIMARY KEY, USR_SEQ INT NOT NULL, APR_TOKEN VARCHAR(255) NOT NULL,
			APR_USED_YN CHAR(1) NOT NULL, EXPIRES_AT DATETIME NOT NULL, REG_DATE DATETIME NOT NULL,
			INDEX IDX_RESET_TOKEN (APR_TOKEN), INDEX IDX_RESET_ACCOUNT (USR_SEQ)
		) ENGINE=InnoDB;
		CREATE TABLE AUTH_IDENTITY (
			IDENTITY_ID BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY, ACCOUNT_ID INT NOT NULL,
			PROVIDER VARCHAR(32) NOT NULL, STATUS VARCHAR(16) NOT NULL
		) ENGINE=InnoDB;
		CREATE TABLE AUTH_PASSWORD_CREDENTIAL (
			IDENTITY_ID BIGINT UNSIGNED PRIMARY KEY, PROVIDER VARCHAR(32) NOT NULL, ALGORITHM VARCHAR(32) NOT NULL,
			PARAMETERS_TEXT VARCHAR(255) NULL, PASSWORD_HASH VARCHAR(255) NOT NULL, STATUS VARCHAR(16) NOT NULL,
			CREATED_AT DATETIME NOT NULL, UPDATED_AT DATETIME NOT NULL
		) ENGINE=InnoDB;
		CREATE TABLE ALUMNI_MOBILE_REFRESH_TOKEN (
			USR_SEQ INT NOT NULL, REVOKED_AT DATETIME NULL, MRT_REVOKED_AT DATETIME NULL
		) ENGINE=InnoDB;
		CREATE TABLE AUTH_SESSION_FAMILY (
			ACCOUNT_ID INT NOT NULL, STATUS VARCHAR(16) NOT NULL, REVOKED_AT DATETIME NULL,
			REVOKE_REASON_CODE VARCHAR(64) NULL, UPDATED_AT DATETIME NOT NULL
		) ENGINE=InnoDB;
		INSERT INTO WEO_MEMBER VALUES (42, 'legacy-before');
		INSERT INTO AUTH_IDENTITY (IDENTITY_ID, ACCOUNT_ID, PROVIDER, STATUS) VALUES (101, 42, 'LOCAL_USERNAME', 'ACTIVE');
		INSERT INTO AUTH_PASSWORD_CREDENTIAL VALUES (101, 'LOCAL_USERNAME', 'ARGON2ID', NULL, 'canonical-before', 'ACTIVE', NOW(), NOW());
		INSERT INTO ALUMNI_PASSWORD_RESET (USR_SEQ, APR_TOKEN, APR_USED_YN, EXPIRES_AT, REG_DATE) VALUES
			(42, 'first-reset-token', 'N', DATE_ADD(NOW(), INTERVAL 15 MINUTE), NOW()),
			(42, 'second-reset-token', 'N', DATE_ADD(NOW(), INTERVAL 15 MINUTE), NOW());
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func passwordResetIntegrationCredential(hash string) model.PasswordCredential {
	parameters := "m=19456,t=2,p=1"
	return model.PasswordCredential{
		Provider: model.IdentityProviderLocalUsername, Algorithm: model.PasswordAlgorithmArgon2id,
		ParametersText: &parameters, PasswordHash: hash, Status: model.PasswordCredentialStatusActive,
	}
}
