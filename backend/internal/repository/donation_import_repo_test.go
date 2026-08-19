package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestFindMemberCandidatesByNameCohortPhoneReturnsEveryActiveExactMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT USR_SEQ, USR_NAME[\s\S]*USR_NAME = \?[\s\S]*USR_FN = \?[\s\S]*USR_STATUS != 'AAA'`).
		WithArgs("김동문", "11", "01012345678", "01012345678").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_NAME"}).
			AddRow(11, "김동문").
			AddRow(19, "김동문"))

	candidates, err := repo.FindMemberCandidatesByNameCohortPhone(" 김동문 ", " 11 ", "010-1234-5678")
	if err != nil {
		t.Fatalf("FindMemberCandidatesByNameCohortPhone() error = %v", err)
	}
	if len(candidates) != 2 || candidates[0].USRSeq != 11 || candidates[1].USRSeq != 19 {
		t.Fatalf("candidates = %+v, want both matches", candidates)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindMemberCandidatesByKeysUsesSingleBatchQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))
	keys := []model.DonationImportMemberKey{
		model.NewDonationImportMemberKey("김동문", "11", "010-1111-1111"),
		model.NewDonationImportMemberKey("이동문", "12", "010-2222-2222"),
	}

	mock.ExpectQuery(`SELECT requested.MATCH_NAME,[\s\S]*FROM \(SELECT \? AS MATCH_NAME,[\s\S]*UNION ALL SELECT \? AS MATCH_NAME,[\s\S]*JOIN WEO_MEMBER`).
		WithArgs("김동문", "11", "01011111111", "이동문", "12", "01022222222").
		WillReturnRows(sqlmock.NewRows([]string{"MATCH_NAME", "MATCH_COHORT", "MATCH_PHONE", "USR_SEQ", "USR_NAME"}).
			AddRow("김동문", "11", "01011111111", 11, "김동문").
			AddRow("이동문", "12", "01022222222", 12, "이동문"))

	result, err := repo.FindMemberCandidatesByKeys(keys)
	if err != nil {
		t.Fatal(err)
	}
	if len(result[keys[0]]) != 1 || result[keys[0]][0].USRSeq != 11 || len(result[keys[1]]) != 1 || result[keys[1]][0].USRSeq != 12 {
		t.Fatalf("batch candidates = %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindMemberCandidatesByKeysPreservesDatabaseCaseInsensitiveMatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))
	key := model.NewDonationImportMemberKey("Alice", "12", "010-1111-1111")

	mock.ExpectQuery(`SELECT requested.MATCH_NAME,[\s\S]*JOIN WEO_MEMBER`).
		WithArgs("Alice", "12", "01011111111").
		WillReturnRows(sqlmock.NewRows([]string{"MATCH_NAME", "MATCH_COHORT", "MATCH_PHONE", "USR_SEQ", "USR_NAME"}).
			AddRow("Alice", "12", "01011111111", 11, "Alice").
			AddRow("Alice", "12", "01011111111", 19, "ALICE"))

	result, err := repo.FindMemberCandidatesByKeys([]model.DonationImportMemberKey{key})
	if err != nil {
		t.Fatal(err)
	}
	if len(result[key]) != 2 || result[key][0].USRSeq != 11 || result[key][1].USRSeq != 19 {
		t.Fatalf("case-insensitive candidates = %+v, want both database matches", result[key])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindMemberCandidatesByKeysTxLocksBatchMatchesForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := repository.NewDonationImportRepository(sqlxDB)
	key := model.NewDonationImportMemberKey("김동문", "11", "010-1111-1111")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT requested.MATCH_NAME,[\s\S]*ORDER BY member.USR_SEQ FOR UPDATE`).
		WithArgs("김동문", "11", "01011111111").
		WillReturnRows(sqlmock.NewRows([]string{"MATCH_NAME", "MATCH_COHORT", "MATCH_PHONE", "USR_SEQ", "USR_NAME"}).
			AddRow("김동문", "11", "01011111111", 11, "김동문"))
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	result, err := repo.FindMemberCandidatesByKeysTx(tx, []model.DonationImportMemberKey{key})
	if err != nil {
		t.Fatal(err)
	}
	if len(result[key]) != 1 || result[key][0].USRSeq != 11 {
		t.Fatalf("locked candidates = %+v", result[key])
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindExistingCompositeKeysUsesSingleBatchQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewDonationImportRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT O_COMPOSITE_KEY FROM WEO_ORDER WHERE O_COMPOSITE_KEY IN \(\?,\?\)`).
		WithArgs("key-1", "key-2").
		WillReturnRows(sqlmock.NewRows([]string{"O_COMPOSITE_KEY"}).AddRow("key-2"))

	result, err := repo.FindExistingCompositeKeys([]string{"key-1", "key-2"})
	if err != nil || result["key-1"] || !result["key-2"] {
		t.Fatalf("FindExistingCompositeKeys() = %+v, %v", result, err)
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
