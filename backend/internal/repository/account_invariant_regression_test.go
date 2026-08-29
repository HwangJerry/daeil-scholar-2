package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestIsApprovedAlumniRequiresLoginEligibleMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewMessageRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`FROM ALUMNI_VERIFICATION v[\s\S]*JOIN WEO_MEMBER m[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_STATUS IN \('CCC','ZZZ'\)`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"EXISTS"}).AddRow(false))

	approved, err := repo.IsApprovedAlumni(42)
	if err != nil || approved {
		t.Fatalf("approved = %v, err = %v", approved, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePasswordAccountClassifiesConcurrentPhoneClaim(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSignupRepository(sqlx.NewDb(db, "sqlmock"))
	request, credential := passwordSignupFixture()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_PHONE_CLAIM`).
		WithArgs(request.Phone, 42).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate phone"})
	mock.ExpectRollback()

	_, err = repo.CreatePasswordAccount(request, "legacy-hash", credential)
	if !errors.Is(err, ErrPhoneAlreadyClaimed) {
		t.Fatalf("error = %v, want ErrPhoneAlreadyClaimed", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppleEmailPreferenceUpdateCannotReactivateDisconnectingLink(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`SET NMS_EMAIL_ENABLED = \?[\s\S]*NMS_STATUS = 'ACTIVE'`).
		WithArgs("N", 42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateSocialProviderEmailEnabled(42, "AP", false); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAnonymizeAccountForDeletionUpdatesOnlyMemberStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`^UPDATE WEO_MEMBER SET USR_STATUS = 'AAA' WHERE USR_SEQ = \?$`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.AnonymizeAccountForDeletion(42); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAnonymizeAccountForDeletionRequiresExactlyOneMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`^UPDATE WEO_MEMBER SET USR_STATUS = 'AAA' WHERE USR_SEQ = \?$`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.AnonymizeAccountForDeletion(42); err == nil {
		t.Fatal("account deletion must fail unless exactly one member is updated")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
