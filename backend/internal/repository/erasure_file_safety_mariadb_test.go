// erasure_file_safety_mariadb_test.go — Real queue isolation and external URL handoff regressions.
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"testing"
)

func TestErasureFileSafetyOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("set AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(200),USR_ID VARCHAR(100),USR_EMAIL VARCHAR(100),USR_PHONE VARCHAR(100)) ENGINE=InnoDB;
CREATE TABLE WEO_MEMBER_SOCIAL (USR_SEQ INT,NMS_GATE CHAR(2),NMS_ID VARCHAR(100)) ENGINE=InnoDB;
CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,CONTENTS TEXT) ENGINE=InnoDB;
CREATE TABLE ALUMNI_UPLOAD_OWNER (F_SEQ INT,USR_SEQ INT,URL_PATH VARCHAR(200)) ENGINE=InnoDB;
CREATE TABLE ALUMNI_ERASURE_FILE (ID INT AUTO_INCREMENT PRIMARY KEY,REQUEST_ID BIGINT,URL_PATH VARCHAR(200),URL_HASH VARCHAR(64)) ENGINE=InnoDB;
CREATE TABLE ALUMNI_ACCOUNT_ERASURE (REQUEST_ID BIGINT PRIMARY KEY,MODE VARCHAR(30),STAGE VARCHAR(30)) ENGINE=InnoDB;
CREATE TABLE WEO_FILES (F_SEQ INT PRIMARY KEY,F_GATE CHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(200),FILE_NAME VARCHAR(100)) ENGINE=InnoDB;
INSERT INTO WEO_MEMBER VALUES (42,'','a','a@example.org',''),(43,'/files/shared.jpg','b','b@example.org','');
INSERT INTO WEO_BOARDBBS VALUES (1,42,'<img src="/old/upload/shared.jpg">');
INSERT INTO ALUMNI_UPLOAD_OWNER VALUES (7,43,'/files/shared.jpg');
INSERT INTO WEO_FILES VALUES (7,'PF',43,'/files','shared.jpg');
INSERT INTO ALUMNI_ACCOUNT_ERASURE VALUES (1,'automatic','database_erased');`)
	w := model.ErasureWork{RequestID: 1, UserSeq: 42}
	origin := "https://app.example.org"
	for _, photo := range []string{"", "https://app.example.org/upload/shared.jpg"} {
		db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO=? WHERE USR_SEQ=42`, photo)
		tx, err := db.Beginx()
		if err != nil {
			t.Fatal(err)
		}
		schema, err := readErasureSchema(tx)
		if err != nil {
			t.Fatal(err)
		}
		err = queueErasureFiles(tx, schema, w, origin)
		if err == nil {
			t.Fatal("another member's file accepted")
		}
		tx.Rollback()
	}
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM WEO_FILES WHERE F_SEQ=7`); err != nil || n != 1 {
		t.Fatal("other file metadata lost", err)
	}
	// A stale queue still cannot unlink a file referenced after the member transaction.
	db.MustExec(`INSERT INTO ALUMNI_ERASURE_FILE (ID,REQUEST_ID,URL_PATH,URL_HASH) VALUES (1,1,'/upload/shared.jpg','test')`)
	repo := &AccountDeletionRequestRepository{DB: db, SiteOrigin: origin}
	called := false
	if err := repo.EraseFileIfUnreferenced(w, model.ErasureFile{ID: 1, URL: "/upload/shared.jpg"}, func(string) error { called = true; return nil }); err == nil || called {
		t.Fatal("shared queued file unlinked")
	}
	db.MustExec(`DELETE FROM ALUMNI_ERASURE_FILE; DELETE FROM WEO_BOARDBBS; UPDATE WEO_MEMBER SET USR_PHOTO='https://profile.example.net/avatar.jpg' WHERE USR_SEQ=42`)
	subject, err := repo.ErasureExternalSubject(w)
	if err != nil || len(subject.ExternalFileURLs) != 1 || subject.ExternalFileURLs[0] != "https://profile.example.net/avatar.jpg" {
		t.Fatal("external avatar handoff lost", err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	schema, err := readErasureSchema(tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = queueErasureFiles(tx, schema, w, origin); err != nil {
		t.Fatal(err)
	}
	if err = tx.Get(&n, `SELECT COUNT(*) FROM ALUMNI_ERASURE_FILE`); err != nil || n != 0 {
		t.Fatal("external avatar entered local queue", err)
	}
	tx.Rollback()
	// Owned, unreferenced local uploads still progress and acknowledge the queue.
	db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO='/uploads/owned.jpg' WHERE USR_SEQ=42`)
	tx, err = db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err = queueErasureFiles(tx, schema, w, origin); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DELETE FROM WEO_MEMBER WHERE USR_SEQ=42`)
	files, err := repo.ErasureFiles(1)
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	if err = repo.EraseFileIfUnreferenced(w, files[0], func(path string) error {
		called = true
		if path != "/uploads/owned.jpg" {
			t.Fatal(path)
		}
		return nil
	}); err != nil || !called {
		t.Fatal("owned file blocked", err)
	}
}
