package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestInsertMobileRefreshTokenUsesCanonicalVerificationRelation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ`).
		WithArgs("new-jti", "family", sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.InsertMobileRefreshToken(42, "family", "new-jti", time.Now().Add(time.Hour), "CCC"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateMobileRefreshTokenRevokesSessionFamilyOnReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT`).
		WithArgs("old-jti", 42).
		WillReturnRows(sqlmock.NewRows([]string{"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT"}).
			AddRow("family", now.Add(time.Hour), now, nil))
	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42, "family").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err = repo.RotateMobileRefreshToken(42, "family", "old-jti", "new-jti", now.Add(time.Hour), "CCC")
	if !errors.Is(err, ErrRefreshTokenReplay) {
		t.Fatalf("rotation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateMobileRefreshTokenCommitsExactlyOneSuccessor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT, MRT_REVOKED_AT`).
		WithArgs("old-jti", 42).
		WillReturnRows(sqlmock.NewRows([]string{
			"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT", "MRT_REVOKED_AT",
		}).AddRow("family", now.Add(time.Hour), nil, nil, nil))
	mock.ExpectExec(`SET CONSUMED_AT = NOW\(\), ROTATED_TO_JTI = \?`).
		WithArgs("new-jti", "old-jti", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT \?, m.USR_SEQ, \?, \?, NOW\(\)`).
		WithArgs("new-jti", "family", sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := repo.RotateMobileRefreshToken(42, "family", "old-jti", "new-jti", now.Add(time.Hour), "CCC"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRotateMobileRefreshTokenRollsBackWhenPrincipalChangesAtSuccessorInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT, MRT_REVOKED_AT`).
		WithArgs("old-jti", 42).
		WillReturnRows(sqlmock.NewRows([]string{
			"MRT_SID", "EXPIRES_AT", "CONSUMED_AT", "REVOKED_AT", "MRT_REVOKED_AT",
		}).AddRow("family", now.Add(time.Hour), nil, nil, nil))
	mock.ExpectExec(`SET CONSUMED_AT = NOW\(\), ROTATED_TO_JTI = \?`).
		WithArgs("new-jti", "old-jti", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT \?, m.USR_SEQ, \?, \?, NOW\(\)`).
		WithArgs("new-jti", "family", sqlmock.AnyArg(), 42, "CCC").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.RotateMobileRefreshToken(42, "family", "old-jti", "new-jti", now.Add(time.Hour), "CCC")
	if !errors.Is(err, ErrSessionPrincipalChanged) {
		t.Fatalf("rotation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
