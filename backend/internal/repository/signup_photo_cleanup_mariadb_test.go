package repository

import (
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"testing"
	"time"
)

func TestSocialSignupPhotoOwnershipMariaDB(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_FILES(F_SEQ INT PRIMARY KEY,F_GATE VARCHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(255),FILE_NAME VARCHAR(255)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_UPLOAD_OWNER(F_SEQ INT,USR_SEQ INT,URL_PATH VARCHAR(500)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_PROFILE_FILE_HISTORY(USR_SEQ INT,URL_PATH VARCHAR(500)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(500)) ENGINE=InnoDB;
 INSERT INTO WEO_FILES VALUES(1,'PR',0,'/uploads/profile','orphan.jpg'),(2,'PR',0,'/uploads/profile','committed.jpg'),(3,'PR',99,'/uploads/profile','joined.jpg'),(4,'PR',0,'/uploads/profile','owned.jpg'),(5,'PR',0,'/uploads/profile','previous.jpg');
 INSERT INTO WEO_MEMBER VALUES(42,'https://app.example.test/uploads/profile/committed.jpg');
 INSERT INTO ALUMNI_UPLOAD_OWNER VALUES(4,43,'/uploads/profile/owned.jpg');
 INSERT INTO ALUMNI_PROFILE_FILE_HISTORY VALUES(44,'https://app.example.test/uploads/profile/previous.jpg');
 `)).DB
	repo := NewFileRepository(db)
	for _, tc := range []struct {
		ID      int
		URL     string
		Discard bool
	}{{2, "/uploads/profile/committed.jpg", false}, {3, "/uploads/profile/joined.jpg", false}, {4, "/uploads/profile/owned.jpg", false}, {5, "/uploads/profile/previous.jpg", false}, {1, "https://provider.example/photo", false}, {1, "/uploads/profile/orphan.jpg", true}} {
		called := false
		err := repo.DiscardSignupProfileUpload(tc.ID, tc.URL, "https://app.example.test", func(string) error { called = true; return nil })
		if err != nil || called != tc.Discard {
			t.Fatalf("id=%d erased=%v want=%v error=%v", tc.ID, called, tc.Discard, err)
		}
	}
}

func TestSocialPhotoDiscardHoldsReferenceLockThroughUnlink(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_FILES(F_SEQ INT PRIMARY KEY,F_GATE VARCHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(255),FILE_NAME VARCHAR(255)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(500)) ENGINE=InnoDB;
 INSERT INTO WEO_FILES VALUES(1,'PR',0,'/uploads/profile','orphan.jpg');
 INSERT INTO WEO_MEMBER VALUES(42,'');
 `)).DB
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- NewFileRepository(db).DiscardSignupProfileUpload(1, "/uploads/profile/orphan.jpg", "https://app.example.test", func(string) error { close(entered); <-release; return nil })
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("discard did not reach unlink")
	}
	updated := make(chan error, 1)
	go func() {
		_, err := db.Exec(`UPDATE WEO_MEMBER SET USR_PHOTO='https://app.example.test/uploads/profile/other.jpg' WHERE USR_SEQ=42`)
		updated <- err
	}()
	select {
	case err := <-updated:
		close(release)
		t.Fatalf("member reference write ran during unlink: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-updated:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reference lock not released")
	}
}
