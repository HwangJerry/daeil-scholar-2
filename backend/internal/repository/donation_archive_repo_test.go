package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"testing"
)

func TestArchiveReadRequiresCommittedAudit(t *testing.T) {
	for _, mode := range []string{"success", "expired", "decrypt_failure", "audit_failure", "commit_failure"} {
		t.Run(mode, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			r := DonationArchiveRepository{DB: sqlx.NewDb(db, "sqlmock")}
			mock.ExpectBegin()
			q := mock.ExpectQuery(`(?s)SELECT CIPHERTEXT.*RETAIN_UNTIL >=.*FOR UPDATE`).WithArgs(int64(7))
			if mode == "expired" {
				q.WillReturnError(sql.ErrNoRows)
			} else {
				q.WillReturnRows(sqlmock.NewRows([]string{"CIPHERTEXT"}).AddRow([]byte("encrypted")))
			}
			if mode == "expired" || mode == "decrypt_failure" {
				mock.ExpectRollback()
			} else {
				e := mock.ExpectExec(`INSERT INTO ALUMNI_DONATION_ARCHIVE_ACCESS`).WithArgs(int64(7), 42, "accounting_review")
				if mode == "audit_failure" {
					e.WillReturnError(errors.New("audit unavailable"))
					mock.ExpectRollback()
				} else {
					e.WillReturnResult(sqlmock.NewResult(1, 1))
					c := mock.ExpectCommit()
					if mode == "commit_failure" {
						c.WillReturnError(errors.New("commit failed"))
					}
				}
			}
			result, err := r.Read(context.Background(), 7, 42, "accounting_review", func([]byte) ([]byte, error) {
				if mode == "decrypt_failure" {
					return nil, errors.New("bad key")
				}
				return []byte(`{"donorName":"synthetic"}`), nil
			})
			if mode == "success" {
				if err != nil || len(result) == 0 {
					t.Fatal(err)
				}
			} else if err == nil || len(result) != 0 {
				t.Fatal("plaintext released on failure")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
