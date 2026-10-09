package service

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"os"
	"path/filepath"
	"testing"
)

func expectRetiredUpload(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(F_GATE`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"F_GATE", "F_JOIN_SEQ", "URL_PATH"}).AddRow("PR", 0, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg"))
	mock.ExpectQuery(`information_schema.COLUMNS`).WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "COLUMN_NAME"}).AddRow("WEO_FILES", "F_SEQ").AddRow("WEO_FILES", "FILE_PATH").AddRow("WEO_FILES", "FILE_NAME"))
	mock.ExpectQuery(`SELECT CONCAT\(FILE_PATH`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"URL"}))
	mock.ExpectExec(`DELETE FROM WEO_FILES`).WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}

func TestRetiredSignupPhotoUnlinkFailureRetriesWithExactCommittedReceipt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	root := t.TempDir()
	target := filepath.Join(root, "profile", "aaaaaaaaaaaaaaaaaaaaaaaa.jpg")
	if err := os.MkdirAll(target, 0700); err != nil {
		t.Fatal(err)
	}
	// A nonempty directory at the uploaded file path injects a real unlink failure.
	child := filepath.Join(target, "synthetic-blocker")
	if err := os.WriteFile(child, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	uploader := NewUploadOrchestrator(NewFileStorageService(root), nil, NewFileRecordService(repository.NewFileRepository(sqlx.NewDb(db, "sqlmock"))))
	uploader.SetSiteOrigin("https://app.example.test")
	expectRetiredUpload(mock)
	result := &UploadResult{FSeq: 1, URL: "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg"}
	if err := uploader.DiscardUnclaimedProfile(result); err == nil {
		t.Fatal("unlink fault did not fail")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("metadata was not retired before unlink: %v", err)
	}
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	// No database expectation here: the committed receipt must authorize retry.
	if err := uploader.DiscardUnclaimedProfile(result); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("receipt retry did not unlink actual tracked path: %v", err)
	}
}

func TestMissingSignupUploadRowDoesNotAuthorizeArbitraryDiskErase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	root := t.TempDir()
	target := filepath.Join(root, "profile", "untracked.jpg")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	uploader := NewUploadOrchestrator(NewFileStorageService(root), nil, NewFileRecordService(repository.NewFileRepository(sqlx.NewDb(db, "sqlmock"))))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(F_GATE`).WithArgs(404).WillReturnRows(sqlmock.NewRows([]string{"F_GATE", "F_JOIN_SEQ", "URL_PATH"}))
	mock.ExpectRollback()
	if err := uploader.DiscardUnclaimedProfile(&UploadResult{FSeq: 404, URL: "/uploads/profile/untracked.jpg"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("missing metadata authorized arbitrary unlink", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRetiredSignupPhotoUnlinkReceiptIsBounded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	root := t.TempDir()
	target := filepath.Join(root, "profile", "aaaaaaaaaaaaaaaaaaaaaaaa.jpg")
	if err := os.MkdirAll(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "blocker"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	uploader := NewUploadOrchestrator(NewFileStorageService(root), nil, NewFileRecordService(repository.NewFileRepository(sqlx.NewDb(db, "sqlmock"))))
	expectRetiredUpload(mock)
	result := &UploadResult{FSeq: 1, URL: "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg"}
	for attempt := 0; attempt < signupPhotoMaxUnlinkAttempts; attempt++ {
		if err := uploader.DiscardUnclaimedProfile(result); err == nil {
			t.Fatal("unlink fault did not fail")
		}
	}
	if len(uploader.signupRetirements) != 0 {
		t.Fatal("exhausted receipt retained")
	}
	// Even once disk unlink becomes possible, a fifth call must not reconstruct
	// deletion permission from the now-missing row.
	if err := os.Remove(filepath.Join(target, "blocker")); err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COALESCE\(F_GATE`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"F_GATE", "F_JOIN_SEQ", "URL_PATH"}))
	mock.ExpectRollback()
	if err := uploader.DiscardUnclaimedProfile(result); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("exhausted receipt reconstructed unlink permission", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
