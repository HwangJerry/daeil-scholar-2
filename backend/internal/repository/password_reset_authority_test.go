package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestFindPasswordResetAccountByVerifiedEmailUsesCanonicalAuthority(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewPasswordResetRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`FROM AUTH_IDENTITY identity[\s\S]*JOIN AUTH_ACCOUNT_STATE[\s\S]*identity\.PROVIDER = 'EMAIL'[\s\S]*identity\.STATUS = 'ACTIVE'[\s\S]*identity\.VERIFIED_AT IS NOT NULL[\s\S]*account_state\.STATUS = 'ACTIVE'`).
		WithArgs("member@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO", "REG_DATE",
		}).AddRow(42, "member", "Member", "CCC", "", "", "profile@example.net", "", "", time.Now()))

	user, err := repo.FindPasswordResetAccountByVerifiedEmail("member@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user == nil || user.USRSeq != 42 {
		t.Fatalf("user = %+v", user)
	}
}
