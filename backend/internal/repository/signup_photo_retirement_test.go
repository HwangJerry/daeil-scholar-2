package repository

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"testing"
)

func TestSignupPhotoDeleteOrCommitFailureNeverUnlinks(t *testing.T) {
	for _, stage := range []string{"delete", "commit"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewFileRepository(sqlx.NewDb(db, "sqlmock"))
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT COALESCE\(F_GATE`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"F_GATE", "F_JOIN_SEQ", "URL_PATH"}).AddRow("PR", 0, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg"))
			mock.ExpectQuery(`information_schema.COLUMNS`).WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "COLUMN_NAME"}).AddRow("WEO_FILES", "F_SEQ").AddRow("WEO_FILES", "FILE_PATH").AddRow("WEO_FILES", "FILE_NAME"))
			mock.ExpectQuery(`SELECT CONCAT\(FILE_PATH`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"URL"}))
			if stage == "delete" {
				mock.ExpectExec(`DELETE FROM WEO_FILES`).WithArgs(1).WillReturnError(errors.New("synthetic delete failure"))
				mock.ExpectRollback()
			} else {
				mock.ExpectExec(`DELETE FROM WEO_FILES`).WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("synthetic indeterminate commit failure"))
			}
			erased := 0
			err = repo.DiscardSignupProfileUpload(1, "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg", "https://app.example.test", func(string) error { erased++; return nil })
			if err == nil {
				t.Fatal("fault was not returned")
			}
			if erased != 0 {
				t.Fatalf("%s failure already unlinked file %d times", stage, erased)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
