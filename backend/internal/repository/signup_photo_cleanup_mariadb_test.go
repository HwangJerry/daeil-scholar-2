package repository

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
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
		var blocked *model.ErasureBlocked
		if (err != nil && !errors.As(err, &blocked)) || called != tc.Discard {
			t.Fatalf("id=%d erased=%v want=%v error=%v", tc.ID, called, tc.Discard, err)
		}
	}
}

func TestSocialPhotoDiscardDoesNotBlockUnrelatedMemberWrite(t *testing.T) {
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
		if err != nil {
			close(release)
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("unrelated member update blocked by signup cleanup")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSignupPhotoAliasAndReviewBlockScopeMariaDB(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_FILES(F_SEQ INT PRIMARY KEY,F_GATE VARCHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(255),FILE_NAME VARCHAR(255)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_PHOTO VARCHAR(500)) ENGINE=InnoDB;
 CREATE TABLE WEO_BOARDBBS(SEQ INT PRIMARY KEY,USR_SEQ INT,CONTENTS TEXT) ENGINE=InnoDB;
 INSERT INTO WEO_FILES VALUES(1,'PR',0,'/uploads/profile','aaaaaaaaaaaaaaaaaaaaaaaa.jpg');
 INSERT INTO WEO_MEMBER VALUES(42,'');
 `)).DB
	repo := NewFileRepository(db)
	for _, raw := range []string{"/uploads/profile/%61aaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://www.app.example.test/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg?version=1#preview", "https:/uploads/profile/%61aaaaaaaaaaaaaaaaaaaaaaa.jpg?bad=%zz"} {
		db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO=? WHERE USR_SEQ=42`, raw)
		erased := false
		err := repo.DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { erased = true; return nil })
		var blocked *model.ErasureBlocked
		if !errors.As(err, &blocked) || erased {
			t.Fatalf("existing alias not preserved: erased=%v error=%v", erased, err)
		}
	}
	db.MustExec(`UPDATE WEO_MEMBER SET USR_PHOTO='https:/files/unrelated.jpg' WHERE USR_SEQ=42`)
	db.MustExec(`INSERT INTO WEO_BOARDBBS VALUES(1,42,TO_BASE64('<img src="/uploads/profile/%61aaaaaaaaaaaaaaaaaaaaaaa.jpg">'))`)
	erased := false
	err := repo.DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { erased = true; return nil })
	var blocked *model.ErasureBlocked
	if !errors.As(err, &blocked) || erased {
		t.Fatalf("base64 content reference missed: %v", err)
	}
	db.MustExec(`DELETE FROM WEO_BOARDBBS`)
	err = repo.DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { erased = true; return nil })
	if err != nil || !erased {
		t.Fatalf("unrelated malformed URL globally blocked cleanup: %v", err)
	}
}

func TestSignupPhotoRetiredMetadataRejectsLateClaimBeforeDiskErase(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_FILES(F_SEQ INT PRIMARY KEY,F_GATE VARCHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(255),FILE_NAME VARCHAR(255)) ENGINE=InnoDB;
 CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3),USR_PHOTO VARCHAR(500)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_UPLOAD_OWNER(F_SEQ INT PRIMARY KEY,USR_SEQ INT,URL_PATH VARCHAR(500)) ENGINE=InnoDB;
 INSERT INTO WEO_FILES VALUES(1,'PR',0,'/uploads/profile','aaaaaaaaaaaaaaaaaaaaaaaa.jpg');
 INSERT INTO WEO_MEMBER VALUES(42,'CCC','');
 `)).DB
	entered := make(chan struct{})
	release := make(chan struct{})
	discardDone := make(chan error, 1)
	go func() {
		discardDone <- NewFileRepository(db).DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { close(entered); <-release; return nil })
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("discard did not reach unlink")
	}
	claimDone := make(chan error, 1)
	go func() {
		claimDone <- (&ProfileRepository{DB: db}).AssignProfileUpload(42, 1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", false)
	}()
	select {
	case err := <-claimDone:
		if !errors.Is(err, ErrManagedUploadUnavailable) {
			close(release)
			t.Fatalf("claim after metadata retirement=%v", err)
		}
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("late claim blocked on disk deletion instead of rejecting retired metadata")
	}
	close(release)
	if err := <-discardDone; err != nil {
		t.Fatal(err)
	}

	var owners int
	_ = db.Get(&owners, `SELECT COUNT(*) FROM ALUMNI_UPLOAD_OWNER`)
	var photo string
	_ = db.Get(&photo, `SELECT USR_PHOTO FROM WEO_MEMBER WHERE USR_SEQ=42`)
	if owners != 0 || photo != "" {
		t.Fatal("failed claim wrote dangling ownership/profile")
	}
}

func TestSignupPhotoDeleteFailureMariaDBLeavesFileAndMetadataIntact(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_FILES(F_SEQ INT PRIMARY KEY,F_GATE VARCHAR(2),F_JOIN_SEQ INT,FILE_PATH VARCHAR(255),FILE_NAME VARCHAR(255)) ENGINE=InnoDB;
 INSERT INTO WEO_FILES VALUES(1,'PR',0,'/uploads/profile','aaaaaaaaaaaaaaaaaaaaaaaa.jpg');
 CREATE TRIGGER synthetic_signup_retire_fail BEFORE DELETE ON WEO_FILES FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic metadata retirement failure';
 `)).DB
	erased := 0
	err := NewFileRepository(db).DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { erased++; return nil })
	if err == nil || erased != 0 {
		t.Fatalf("failed DELETE unlinked disk: erased=%d error=%v", erased, err)
	}
	var rows int
	if err := db.Get(&rows, `SELECT COUNT(*) FROM WEO_FILES WHERE F_SEQ=1`); err != nil || rows != 1 {
		t.Fatalf("failed DELETE did not preserve metadata: rows=%d error=%v", rows, err)
	}
}
