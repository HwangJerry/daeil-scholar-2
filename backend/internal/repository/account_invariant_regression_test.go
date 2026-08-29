package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
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

func TestReserveSocialDisconnectSerializesOnMemberAndIgnoresDisabledPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT NMS_STATUS`).WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"NMS_STATUS"}).AddRow("ACTIVE"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i[\s\S]*AUTH_PASSWORD_CREDENTIAL c[\s\S]*i.STATUS = 'ACTIVE'[\s\S]*c.STATUS = 'ACTIVE'`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"HAS_PASSWORD"}).AddRow(false))
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS = 'ACTIVE'`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("KT").AddRow("AP"))
	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL[\s\S]*SET NMS_STATUS = 'DISCONNECTING'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT", 42, "KT").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	connections, phase, err := repo.ReserveSocialDisconnect(42, model.SocialProviderKakao)
	if err != nil || phase != SocialDisconnectRevokeFresh || connections.HasPassword {
		t.Fatalf("connections = %#v, phase = %v, err = %v", connections, phase, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReserveSocialDisconnectResumesPendingReservationAfterRestart(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectQuery(`SELECT NMS_STATUS`).WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"NMS_STATUS"}).AddRow("DISCONNECTING"))
	mock.ExpectQuery(`FROM AUTH_IDENTITY i`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"HAS_PASSWORD"}).AddRow(true))
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS = 'ACTIVE'`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}))
	mock.ExpectCommit()

	_, phase, err := repo.ReserveSocialDisconnect(42, model.SocialProviderKakao)
	if err != nil || phase != SocialDisconnectRevokeRetry {
		t.Fatalf("phase = %v, err = %v", phase, err)
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
