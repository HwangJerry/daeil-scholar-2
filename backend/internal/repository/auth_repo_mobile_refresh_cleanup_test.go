package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestDeleteExpiredMobileRefreshTokens(t *testing.T) {
	revokedBefore := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name         string
		rowsAffected int64
	}{
		{name: "expired token is deleted", rowsAffected: 1},
		{name: "old revoked token is deleted", rowsAffected: 1},
		{name: "recently revoked token is retained", rowsAffected: 0},
		{name: "unexpired consumed token is retained", rowsAffected: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			mock.ExpectExec(`^DELETE FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE EXPIRES_AT < NOW\(\) OR \(REVOKED_AT IS NOT NULL AND REVOKED_AT < \?\)$`).
				WithArgs(revokedBefore).
				WillReturnResult(sqlmock.NewResult(0, testCase.rowsAffected))

			deleted, err := repo.DeleteExpiredMobileRefreshTokens(revokedBefore)
			if err != nil {
				t.Fatal(err)
			}
			if deleted != testCase.rowsAffected {
				t.Fatalf("deleted = %d, want %d", deleted, testCase.rowsAffected)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
