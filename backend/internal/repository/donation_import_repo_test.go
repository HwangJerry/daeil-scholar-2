package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestFindMemberCandidatesByNamePhoneReturnsEveryActiveExactMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT USR_SEQ, USR_NAME[\s\S]*USR_NAME = \?[\s\S]*USR_STATUS != 'AAA'`).
		WithArgs("김동문", "01012345678", "01012345678").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_NAME"}).
			AddRow(11, "김동문").
			AddRow(19, "김동문"))

	candidates, err := repo.FindMemberCandidatesByNamePhone(" 김동문 ", "010-1234-5678")
	if err != nil {
		t.Fatalf("FindMemberCandidatesByNamePhone() error = %v", err)
	}
	if len(candidates) != 2 || candidates[0].USRSeq != 11 || candidates[1].USRSeq != 19 {
		t.Fatalf("candidates = %+v, want both matches", candidates)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExtRefExistsChecksTransactionAndCompositeIdentities(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT COUNT\(\*\)[\s\S]*O_TRANSACTION_NO[\s\S]*O_COMPOSITE_KEY`).
		WithArgs("TX-1", "TX-1", "key-1", "key-1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))

	exists, err := repo.ExtRefExists(" TX-1 ", " key-1 ")
	if err != nil {
		t.Fatalf("ExtRefExists() error = %v", err)
	}
	if !exists {
		t.Fatal("ExtRefExists() = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExtRefExistsSkipsQueryWithoutIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))

	exists, err := repo.ExtRefExists("", "")
	if err != nil || exists {
		t.Fatalf("ExtRefExists() = %v, %v; want false, nil", exists, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
