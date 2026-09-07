// erasure_hostless_reference_regression_test.go — Ambiguous browser URLs cannot authorize shared-file deletion.
package repository

import (
	"errors"
	"os"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestErasureHostlessReferencesOnMariaDB101(t *testing.T) {
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
	const fileURL = "/files/shared.jpg"
	for _, raw := range []string{
		"https:/files/shared.jpg", "https:files/shared.jpg",
		"https:////app.example.org/files/shared.jpg", "HTTP:/files/shared.jpg",
		"///app.example.org/files/shared.jpg", "https://app.example.org\\@external.test/files/shared.jpg",
	} {
		for _, reference := range []struct {
			name, statement string
			html            bool
		}{
			{"banner", `INSERT INTO MAIN_BANNER_AD_IMAGE VALUES (1,1,?)`, false},
			{"HTML", `INSERT INTO WEO_BOARDBBS VALUES (1,43,?,'')`, true},
			{"profile", `INSERT INTO WEO_MEMBER VALUES (43,?)`, false},
			{"thumbnail", `INSERT INTO WEO_BOARDBBS VALUES (1,43,'',?)`, false},
		} {
			t.Run(reference.name+"/"+raw, func(t *testing.T) {
				db.MustExec(`DELETE FROM WEO_MEMBER; DELETE FROM WEO_BOARDBBS; DELETE FROM MAIN_BANNER_AD_IMAGE; DELETE FROM ALUMNI_ERASURE_FILE`)
				db.MustExec(`INSERT INTO WEO_MEMBER VALUES (42,?)`, fileURL)
				value := raw
				if reference.html {
					value = `<img src="` + raw + `">`
				}
				db.MustExec(reference.statement, value)
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
				var inserted int
				countErr := tx.Get(&inserted, `SELECT COUNT(*) FROM ALUMNI_ERASURE_FILE WHERE REQUEST_ID=1`)
				tx.Rollback()
				requireHostlessReferenceReview(t, err)
				if countErr != nil || inserted != 0 {
					t.Fatal("ambiguous reference was queued before rollback", inserted, countErr)
				}
				db.MustExec(`DELETE FROM WEO_MEMBER WHERE USR_SEQ=42`)
				db.MustExec(`INSERT INTO ALUMNI_ERASURE_FILE VALUES (1,1,?,'synthetic')`, fileURL)
				called := false
				err = repo.EraseFileIfUnreferenced(work, model.ErasureFile{ID: 1, URL: fileURL}, func(string) error { called = true; return nil })
				requireHostlessReferenceReview(t, err)
				if called {
					t.Fatal("ambiguous shared reference reached unlink")
				}
				var queued int
				if err = db.Get(&queued, `SELECT COUNT(*) FROM ALUMNI_ERASURE_FILE WHERE ID=1`); err != nil || queued != 1 {
					t.Fatal("blocked queue lost", queued, err)
				}
			})
		}
	}
}

func requireHostlessReferenceReview(t *testing.T, err error) {
	t.Helper()
	var blocked *model.ErasureBlocked
	if !errors.As(err, &blocked) || blocked.Code != "FILE_PATH_REVIEW_REQUIRED" {
		t.Fatalf("ambiguous reference not blocked for review: %v", err)
	}
}
