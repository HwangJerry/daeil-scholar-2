// erasure_banner_entity_regression_test.go — Shared banner and literal entity filenames survive erasure.
package repository

import (
	"errors"
	"os"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestErasureBannerAndEntityReferencesOnMariaDB101(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("explicit local Docker integration required")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,CONTENTS TEXT,THUMBNAIL_URL VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE MAIN_BANNER_AD_IMAGE (BNI_SEQ INT PRIMARY KEY,BN_SEQ INT,IMAGE_URL VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_UPLOAD_OWNER (F_SEQ INT,USR_SEQ INT,URL_PATH VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ERASURE_FILE (ID INT AUTO_INCREMENT PRIMARY KEY,REQUEST_ID BIGINT,URL_PATH VARCHAR(200),URL_HASH VARCHAR(64)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ACCOUNT_ERASURE (REQUEST_ID BIGINT PRIMARY KEY,MODE VARCHAR(30),STAGE VARCHAR(30)) ENGINE=InnoDB;
 INSERT INTO ALUMNI_ACCOUNT_ERASURE VALUES (1,'automatic','database_erased')`)
	work := model.ErasureWork{RequestID: 1, UserSeq: 42}
	repo := &AccountDeletionRequestRepository{DB: db, SiteOrigin: "https://app.example.org"}
	const fileURL = "/files/a&copy;.jpg"
	for _, c := range []struct{ name, referenceSQL string }{
		{"banner", `INSERT INTO MAIN_BANNER_AD_IMAGE VALUES (1,1,'/files/a&copy;.jpg')`},
		{"HTML", `INSERT INTO WEO_BOARDBBS VALUES (1,43,'<img src="/files/a&amp;copy;.jpg">','')`},
		{"raw profile", `INSERT INTO WEO_MEMBER VALUES (43,'/files/a&copy;.jpg')`},
		{"raw thumbnail", `INSERT INTO WEO_BOARDBBS VALUES (1,43,'','/files/a&copy;.jpg')`},
	} {
		t.Run(c.name, func(t *testing.T) {
			db.MustExec(`DELETE FROM WEO_MEMBER; DELETE FROM WEO_BOARDBBS; DELETE FROM MAIN_BANNER_AD_IMAGE; DELETE FROM ALUMNI_ERASURE_FILE`)
			db.MustExec(`INSERT INTO WEO_MEMBER VALUES (42,?)`, fileURL)
			db.MustExec(c.referenceSQL)
			tx, err := db.Beginx()
			if err != nil {
				t.Fatal(err)
			}
			schema, err := readErasureSchema(tx)
			if err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
			err = queueErasureFiles(tx, schema, work, repo.SiteOrigin)
			tx.Rollback()
			requireSharedReferenceBlocked(t, err)
			// A queued file must also be protected after account rows have gone.
			db.MustExec(`DELETE FROM WEO_MEMBER WHERE USR_SEQ=42`)
			db.MustExec(`INSERT INTO ALUMNI_ERASURE_FILE VALUES (1,1,?,'synthetic')`, fileURL)
			called := false
			err = repo.EraseFileIfUnreferenced(work, model.ErasureFile{ID: 1, URL: fileURL}, func(string) error { called = true; return nil })
			requireSharedReferenceBlocked(t, err)
			if called {
				t.Fatal("shared file reached unlink")
			}
			var queued int
			if err = db.Get(&queued, `SELECT COUNT(*) FROM ALUMNI_ERASURE_FILE WHERE ID=1`); err != nil || queued != 1 {
				t.Fatal("blocked queue lost", queued, err)
			}
		})
	}
	// The owner's matching raw URL and encoded HTML still form one correct candidate.
	db.MustExec(`DELETE FROM WEO_MEMBER; DELETE FROM WEO_BOARDBBS; DELETE FROM MAIN_BANNER_AD_IMAGE; DELETE FROM ALUMNI_ERASURE_FILE`)
	db.MustExec(`INSERT INTO WEO_MEMBER VALUES (42,?)`, fileURL)
	db.MustExec(`INSERT INTO WEO_BOARDBBS VALUES (1,42,'<img src="/files/a&amp;copy;.jpg">',?)`, fileURL)
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	schema, err := readErasureSchema(tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = queueErasureFiles(tx, schema, work, repo.SiteOrigin); err != nil {
		t.Fatal("owned entity filename blocked", err)
	}
	var urls []string
	if err = tx.Select(&urls, `SELECT URL_PATH FROM ALUMNI_ERASURE_FILE`); err != nil || len(urls) != 1 || urls[0] != fileURL {
		t.Fatal("wrong owned candidate", urls, err)
	}
}

func requireSharedReferenceBlocked(t *testing.T, err error) {
	t.Helper()
	var blocked *model.ErasureBlocked
	if !errors.As(err, &blocked) || blocked.Code != "FILE_STILL_REFERENCED" {
		t.Fatalf("shared reference not blocked: %v", err)
	}
}
