// account_erasure_restore_mariadb_test.go — A restored backup cannot bring an erased member back.
package repository

import (
	"database/sql"
	"errors"
	"os"
	"testing"
)

func TestErasureRestoreReapplyOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("set AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3),USR_ID VARCHAR(50),USR_EMAIL VARCHAR(100),USR_PHOTO VARCHAR(200),USR_BIZ_CARD VARCHAR(200),USR_THUMNAIL TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT,NMS_GATE CHAR(2),NMS_ID VARCHAR(100)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE ALUMNI_MESSAGE (AM_SEQ INT PRIMARY KEY,AM_SENDER_SEQ INT,AM_RECVR_SEQ INT,AM_CONTENT TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,SUBJECT VARCHAR(200),CONTENTS TEXT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`)
	for _, name := range []string{"055_create_message_reports.sql", "056_create_account_deletion_requests.sql", "057_create_automatic_account_erasure.sql", "062_create_profile_file_history.sql", "068_create_erasure_restore_guard.sql"} {
		data, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// The state a backup taken before the erasure would bring back.
	db.MustExec(`INSERT INTO WEO_MEMBER VALUES (42,'AAA','fake42','fake42@example.org','','',NULL),(43,'CCC','fake43','fake43@example.org','','',NULL);
INSERT INTO WEO_MEMBER_SOCIAL VALUES (42,'KT','synthetic-kakao');
INSERT INTO ALUMNI_MESSAGE VALUES (1,42,43,'restored'),(2,43,43,'kept');
INSERT INTO WEO_BOARDBBS VALUES (10,42,'restored subject','restored body'),(11,43,'kept subject','kept body');
INSERT INTO ALUMNI_ERASURE_RESTORE_GUARD VALUES (9,42,UTC_TIMESTAMP(),DATE_ADD(UTC_TIMESTAMP(),INTERVAL 35 DAY)),(8,77,DATE_SUB(UTC_TIMESTAMP(),INTERVAL 40 DAY),DATE_SUB(UTC_TIMESTAMP(),INTERVAL 5 DAY));`)
	repo := &AccountDeletionRequestRepository{DB: db}
	guards, err := repo.RestoreGuards()
	if err != nil || len(guards) != 1 || guards[0].RequestID != 9 || !guards[0].Present {
		t.Fatalf("guards = %+v %v", guards, err)
	}
	if _, err = repo.ReapplyErasureAfterRestore(43); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unguarded member touched: %v", err)
	}
	if _, err = repo.ReapplyErasureAfterRestore(42); err != nil {
		t.Fatal(err)
	}
	checks := map[string]int{
		"SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=42":                                           0,
		"SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=43":                                           1,
		"SELECT COUNT(*) FROM ALUMNI_MESSAGE":                                                        1,
		"SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL":                                                     0,
		"SELECT COUNT(*) FROM WEO_BOARDBBS WHERE SEQ=10 AND USR_SEQ=0 AND CONTENTS<>'restored body'": 1,
		"SELECT COUNT(*) FROM WEO_BOARDBBS WHERE SEQ=11 AND CONTENTS='kept body'":                    1,
	}
	for query, want := range checks {
		var got int
		if err = db.Get(&got, query); err != nil || got != want {
			t.Fatalf("%s = %d (%v), want %d", query, got, err, want)
		}
	}
	if guards, err = repo.RestoreGuards(); err != nil || len(guards) != 1 || guards[0].Present {
		t.Fatalf("member still present after re-erasure: %+v %v", guards, err)
	}
	if err = repo.PurgeExpiredRestoreGuards(t.Context()); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err = db.Get(&remaining, `SELECT COUNT(*) FROM ALUMNI_ERASURE_RESTORE_GUARD`); err != nil || remaining != 1 {
		t.Fatalf("guard purge kept %d rows: %v", remaining, err)
	}
}
