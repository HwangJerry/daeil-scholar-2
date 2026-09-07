// erasure_reference_regression_test.go — Reproduce surviving public file references.
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewSurvivingReferencesMustPreventUnlink(t *testing.T) {
	if os.Getenv("AUTOMATIC_ERASURE_DOCKER_INTEGRATION") != "1" {
		t.Skip("explicit local Docker integration required")
	}
	db := startPasswordResetMariaDB101(t)
	db.MustExec(`CREATE TABLE WEO_MEMBER (USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE WEO_BOARDBBS (SEQ INT PRIMARY KEY,USR_SEQ INT,CONTENTS TEXT) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_UPLOAD_OWNER (F_SEQ INT,USR_SEQ INT,URL_PATH VARCHAR(200)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ERASURE_FILE (ID INT PRIMARY KEY,REQUEST_ID BIGINT,URL_PATH VARCHAR(200),URL_HASH VARCHAR(64)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ACCOUNT_ERASURE (REQUEST_ID BIGINT PRIMARY KEY,MODE VARCHAR(30),STAGE VARCHAR(30)) ENGINE=InnoDB;
 INSERT INTO ALUMNI_ACCOUNT_ERASURE VALUES (1,'automatic','database_erased')`)
	cases := []struct {
		name      string
		author    interface{}
		reference string
	}{
		{"query", 43, "/files/shared.jpg?v=2"},
		{"fragment", 43, "/files/shared.jpg#preview"},
		{"public_zero_author", 0, "/files/shared.jpg"},
		{"null_author", nil, "/files/shared.jpg"},
		{"www_alias", 43, "https://www.app.example.org/files/shared.jpg?v=2"},
		{"protocol_relative", 43, "//app.example.org/files/shared.jpg#preview"},
		{"dot_segments", 43, "/files/sub/../shared.jpg"},
		{"relative", 43, "files/shared.jpg"},
		{"encoded_prefix", 43, "/%66iles/shared.jpg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db.MustExec(`DELETE FROM WEO_MEMBER; DELETE FROM WEO_BOARDBBS; DELETE FROM ALUMNI_ERASURE_FILE`)
			db.MustExec(`INSERT INTO WEO_BOARDBBS VALUES (2,?,?)`, c.author, `<img src="`+c.reference+`">`)
			db.MustExec(`INSERT INTO ALUMNI_ERASURE_FILE VALUES (1,1,'/files/shared.jpg','synthetic')`)
			repo := &AccountDeletionRequestRepository{DB: db, SiteOrigin: "https://app.example.org"}
			called := false
			err := repo.EraseFileIfUnreferenced(model.ErasureWork{RequestID: 1, UserSeq: 42}, model.ErasureFile{ID: 1, URL: "/files/shared.jpg"}, func(string) error { called = true; return nil })
			if err == nil || called {
				t.Fatalf("surviving reference ignored: unlink invoked=%v error=%v", called, err)
			}
		})
	}
}

func TestReviewQueryReferenceServesTheSameFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "shared.jpg"), []byte("synthetic image"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"http://app.example.org/shared.jpg", "http://app.example.org/shared.jpg?v=2"} {
		response := httptest.NewRecorder()
		http.FileServer(http.Dir(root)).ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != 200 || response.Body.String() != "synthetic image" {
			t.Fatal("fixture failed", response.Code)
		}
	}
}
