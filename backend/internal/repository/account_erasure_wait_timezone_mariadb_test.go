package repository

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Production uses a Seoul driver location while deletion DATETIME columns store
// UTC wall-clock values. Exercise the real write/read boundary in both locations.
func TestErasureWaitDeadlineRoundTripMariaDB(t *testing.T) {
	server := mariadb.Start(t)
	for _, location := range []string{"UTC", "Asia/Seoul"} {
		t.Run(location, func(t *testing.T) {
			fixture := server.NewDatabase(t, mariadb.Statement(`
CREATE TABLE ALUMNI_ACCOUNT_ERASURE (
 REQUEST_ID BIGINT PRIMARY KEY, MODE VARCHAR(16), STAGE VARCHAR(32)
) ENGINE=InnoDB;
CREATE TABLE ALUMNI_ERASURE_TARGET (
 REQUEST_ID BIGINT, TARGET VARCHAR(32), STATUS VARCHAR(16),
 EVIDENCE_REFERENCE VARCHAR(200) NOT NULL DEFAULT '', LAST_CODE VARCHAR(64) NOT NULL DEFAULT '',
 ATTEMPTS INT NOT NULL DEFAULT 0, LAST_ATTEMPT_AT DATETIME NULL, UPDATED_AT DATETIME NOT NULL,
 WAIT_COUNT BIGINT NOT NULL DEFAULT 0, WAIT_UNTIL DATETIME NULL,
 PRIMARY KEY(REQUEST_ID,TARGET)
) ENGINE=InnoDB;
INSERT INTO ALUMNI_ACCOUNT_ERASURE VALUES(1,'automatic','database_erased');
INSERT INTO ALUMNI_ERASURE_TARGET(REQUEST_ID,TARGET,STATUS,UPDATED_AT)
 VALUES(1,'other_identifiers','pending',UTC_TIMESTAMP());`))
			cfg, err := mysql.ParseDSN(fixture.DSN)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Loc, err = time.LoadLocation(location)
			if err != nil {
				t.Fatal(err)
			}
			db, err := sqlx.Connect("mysql", cfg.FormatDSN())
			if err != nil {
				t.Fatal("connect disposable fixture")
			}
			t.Cleanup(func() { db.Close() })
			repo := &AccountDeletionRequestRepository{DB: db}
			want := time.Date(2026, 10, 9, 19, 3, 2, 0, time.UTC)
			if err := repo.RecordErasureTargets(1, []model.ErasureTarget{{
				Name: "other_identifiers", Status: "pending", Code: model.PhoneVerificationWaitCode,
				WaitCount: 5, WaitUntil: &want,
			}}); err != nil {
				t.Fatal(err)
			}
			targets, err := repo.ErasureTargets(1)
			if err != nil {
				t.Fatal(err)
			}
			if len(targets) != 1 || targets[0].WaitUntil == nil || targets[0].WaitCount != 5 {
				t.Fatalf("missing wait metadata: %+v", targets)
			}
			if !targets[0].WaitUntil.Equal(want) {
				t.Fatalf("wait deadline shifted by driver location: got %s, want %s", targets[0].WaitUntil, want)
			}
		})
	}
}
