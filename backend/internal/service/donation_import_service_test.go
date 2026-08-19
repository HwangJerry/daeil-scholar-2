package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

func TestParseExcelDonationRowsUsesLegacyPositionsFromThirdRow(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"무시", "해피나눔 후원 내역"},
		{"임의 헤더", "회원명", "회차", "소속", "휴대전화", "후원금"},
		{"무시", " 김동문 ", " 12 ", " 총무부 ", "010-1234-5678", "100,000"},
	})

	rows, validationErrors, err := parseExcelDonationRows(workbook, "2026-08-19")
	if err != nil {
		t.Fatalf("parseExcelDonationRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if len(validationErrors) != 0 {
		t.Fatalf("validationErrors = %+v", validationErrors)
	}
	row := rows[0]
	if row.RowIndex != 3 || row.DonorName != "김동문" || row.DonorCohort != "12" || row.DonorDepartment != "총무부" || row.DonorPhone != "01012345678" || row.Amount != 100000 {
		t.Fatalf("row = %+v", row)
	}
}

func TestParseExcelDonationRowsAcceptsProductionWorkbookHeaders(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"", "", "", "", "", "해피나눔, 계좌이체 건만의 합계"},
		{"", "이름", "기수", "과", "연락처", "2025년 6월"},
		{"", "김동문", "12", "영", "010-1234-5678", "100,000"},
	})

	rows, validationErrors, err := parseExcelDonationRows(workbook, "2026-08-19")
	if err != nil || len(validationErrors) != 0 || len(rows) != 1 {
		t.Fatalf("parseExcelDonationRows() = %+v, %+v, %v; want production header accepted", rows, validationErrors, err)
	}
}

func TestParseExcelDonationRowsRejectsAmountAboveLimit(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"", "이름", "기수", "과", "연락처", "2025년 6월"},
		{"", "김동문", "12", "영", "010-1234-5678", donationImportMaxAmount + 1},
	})

	rows, validationErrors, err := parseExcelDonationRows(workbook, "2026-08-19")
	if err != nil || len(rows) != 0 || len(validationErrors) != 1 {
		t.Fatalf("parseExcelDonationRows() = %+v, %+v, %v; want amount limit error", rows, validationErrors, err)
	}
	if validationErrors[0].Code != "AMOUNT_LIMIT_EXCEEDED" {
		t.Fatalf("validation error = %+v", validationErrors[0])
	}
}

func TestParseExcelDonationRowsRejectsRowsWithMissingRequiredFields(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"", "이름", "기수", "학과", "전화", "금액"},
		{"", "", "1", "부서", "010-1111-1111", "10000"},
		{"", "이름", "1", "부서", "", "10000"},
	})

	rows, validationErrors, err := parseExcelDonationRows(workbook, "2026-08-19")
	if err != nil || len(rows) != 0 || len(validationErrors) != 2 {
		t.Fatalf("parseExcelDonationRows() = %+v, %+v, %v; want two row errors", rows, validationErrors, err)
	}
	if validationErrors[0].RowIndex != 3 || validationErrors[0].Field != "donorName" || validationErrors[0].Code != "REQUIRED" {
		t.Fatalf("first validation error = %+v", validationErrors[0])
	}
	if validationErrors[1].RowIndex != 4 || validationErrors[1].Field != "donorPhone" || validationErrors[1].Code != "REQUIRED" {
		t.Fatalf("second validation error = %+v", validationErrors[1])
	}
}

func TestParseExcelDonationRowsRejectsInvalidHeader(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"", "입금일", "기수", "학과", "전화", "금액"},
		{"", "김동문", "11", "영어과", "010-1111-1111", "10000"},
	})

	rows, validationErrors, err := parseExcelDonationRows(workbook, "2026-08-19")
	if rows != nil || validationErrors != nil || err == nil {
		t.Fatalf("parseExcelDonationRows() = %+v, %+v, %v; want header rejection", rows, validationErrors, err)
	}
	if got, want := err.Error(), "2행 B열은 '이름' 헤더가 아닙니다"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestParseExcelDonationRowsRejectsMoreThanMaximumDataRows(t *testing.T) {
	worksheetRows := make([][]interface{}, 0, donationImportMaxRows+3)
	worksheetRows = append(worksheetRows,
		[]interface{}{"제목"},
		[]interface{}{"", "이름", "기수", "학과", "전화", "금액"},
	)
	for index := 0; index <= donationImportMaxRows; index++ {
		worksheetRows = append(worksheetRows, []interface{}{"", fmt.Sprintf("기부자%d", index), "11", "영어과", "010-1111-1111", index + 1})
	}

	rows, validationErrors, err := parseExcelDonationRows(excelWorkbook(t, worksheetRows), "2026-08-19")
	if err != nil {
		t.Fatalf("parseExcelDonationRows() error = %v", err)
	}
	if len(rows) != donationImportMaxRows || len(validationErrors) != 1 {
		t.Fatalf("row count = %d, validationErrors = %+v", len(rows), validationErrors)
	}
	rowError := validationErrors[0]
	if rowError.RowIndex != donationImportMaxRows+3 || rowError.Code != "ROW_LIMIT_EXCEEDED" || !strings.Contains(rowError.Message, "500개") {
		t.Fatalf("row limit error = %+v", rowError)
	}
}

func TestDonationImportPreviewClassifiesEveryStatus(t *testing.T) {
	repo := &donationImportRepositoryStub{
		candidates: map[string][]model.MemberCandidate{
			"매칭|11": {{USRSeq: 11, Name: "매칭"}},
			"모호|12": {{USRSeq: 21, Name: "모호"}, {USRSeq: 22, Name: "모호"}},
			"중복|14": {{USRSeq: 31, Name: "중복"}},
		},
		duplicates: map[string]bool{},
	}
	service := NewDonationImportService(repo, &donationImportOrderCreatorStub{})
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"", "이름", "기수", "부서", "전화", "금액"},
		{"", "매칭", "11", "A", "01011111111", "10000"},
		{"", "모호", "12", "B", "01022222222", "20000"},
		{"", "없음", "13", "C", "01033333333", "30000"},
		{"", "중복", "14", "D", "01044444444", "40000"},
	})
	duplicateRow := model.ImportedDonationRow{
		ExcelDonationRow: model.ExcelDonationRow{RowIndex: 6, DonorName: "중복", DonorCohort: "14", DonorDepartment: "D", DonorPhone: "01044444444", Amount: 40000},
		DonationDate:     "2026-08-19",
	}
	duplicateKey, err := importedDonationCompositeKey(duplicateRow)
	if err != nil {
		t.Fatal(err)
	}
	repo.duplicates[duplicateKey] = true

	result, err := service.ParsePreview(workbook, "2026-08-19")
	if err != nil {
		t.Fatalf("ParsePreview() error = %v", err)
	}
	if result.MatchedCount != 1 || result.AmbiguousCount != 1 || result.UnmatchedCount != 1 || result.DuplicateCount != 1 {
		t.Fatalf("counts = %+v", result)
	}
	wantStatuses := []model.DonationImportRowStatus{
		model.DonationImportRowMatched,
		model.DonationImportRowAmbiguous,
		model.DonationImportRowUnmatched,
		model.DonationImportRowDuplicate,
	}
	for index, wantStatus := range wantStatuses {
		if result.Rows[index].Status != wantStatus {
			t.Fatalf("row %d status = %q, want %q", index, result.Rows[index].Status, wantStatus)
		}
	}
	if result.Rows[0].MatchedUsrSeq == nil || *result.Rows[0].MatchedUsrSeq != 11 || result.Rows[0].MatchedName != "매칭" {
		t.Fatalf("matched row = %+v", result.Rows[0])
	}
	if result.Rows[3].MatchedUsrSeq == nil || result.Rows[3].Status != model.DonationImportRowDuplicate {
		t.Fatalf("duplicate must override match while retaining candidate: %+v", result.Rows[3])
	}
	if result.Rows[0].DonationDate != "2026-08-19" || result.Rows[0].DonorCohort != "11" {
		t.Fatalf("preview row batch data = %+v", result.Rows[0])
	}
}

func TestDonationImportPreviewRejectsDuplicateCompositeKeyInFile(t *testing.T) {
	service := NewDonationImportService(&donationImportRepositoryStub{}, &donationImportOrderCreatorStub{})
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"", "이름", "기수", "학과", "전화", "금액"},
		{"", "중복", "11", "A", "010-1111-1111", "10000"},
		{"", "중복", "11", "A", "01011111111", "10000"},
	})

	result, err := service.ParsePreview(workbook, "2026-08-19")
	if result != nil || err == nil {
		t.Fatalf("ParsePreview() = %+v, %v; want file rejection", result, err)
	}
	var validationError *DonationImportFileValidationError
	if !errors.As(err, &validationError) || len(validationError.Errors) != 1 {
		t.Fatalf("validation error = %#v", err)
	}
	rowError := validationError.Errors[0]
	if rowError.RowIndex != 4 || rowError.Field != "compositeKey" || rowError.Code != "DUPLICATE_IN_FILE" {
		t.Fatalf("row error = %+v", rowError)
	}
}

func TestDonationImportCommitRollsBackWholeBatchOnFailure(t *testing.T) {
	repo := &donationImportRepositoryStub{duplicates: map[string]bool{}}
	creator := &donationImportOrderCreatorStub{sequences: map[string]int64{"성공": 7001}, errors: map[string]error{"실패": errors.New("insert failed")}}
	service := NewDonationImportService(repo, creator)
	accountUsrSeq := 42
	rows := []model.DonationImportCommitRow{commitRow(3, "성공", 20000, &accountUsrSeq), commitRow(4, "실패", 30000, nil)}
	rows[0].ManualOverride = true
	rows[1].ManualOverride = true
	result, err := service.Commit(rows, "2026-08-19", 9, "192.0.2.10")
	if result != nil || err == nil {
		t.Fatalf("Commit() = %+v, %v; want atomic failure", result, err)
	}
	var commitError *DonationImportCommitError
	if !errors.As(err, &commitError) || commitError.RowError.RowIndex != 4 || commitError.RowError.Message != "저장 중 오류가 발생했습니다." {
		t.Fatalf("commit error = %#v", err)
	}
	if !repo.rolledBack || creator.calls != 2 {
		t.Fatalf("rolledBack = %v, creator calls = %d", repo.rolledBack, creator.calls)
	}
}

func TestDonationImportCommitRejectsChangedAutomaticMatchEvenWithManualOverride(t *testing.T) {
	accountUsrSeq := 42
	repo := &donationImportRepositoryStub{candidates: map[string][]model.MemberCandidate{
		"변경|3": {{USRSeq: 99, Name: "변경"}},
	}}
	creator := &donationImportOrderCreatorStub{sequences: map[string]int64{"변경": 7001}}
	cache := &donationCacheInvalidatorStub{}
	service := NewDonationImportService(repo, creator, cache)
	row := commitRow(3, "변경", 20000, &accountUsrSeq)

	result, err := service.Commit([]model.DonationImportCommitRow{row}, "2026-08-19", 9, "192.0.2.10")
	if result != nil || err == nil || !repo.rolledBack || creator.calls != 0 || cache.calls != 0 {
		t.Fatalf("changed match result=%+v err=%v rollback=%v creates=%d cache=%d", result, err, repo.rolledBack, creator.calls, cache.calls)
	}
	var commitError *DonationImportCommitError
	if !errors.As(err, &commitError) || commitError.RowError.Code != "MEMBER_MATCH_CHANGED" {
		t.Fatalf("commit error = %#v", err)
	}

	repo.rolledBack = false
	row.ManualOverride = true
	result, err = service.Commit([]model.DonationImportCommitRow{row}, "2026-08-19", 9, "192.0.2.10")
	if result != nil || err == nil || !repo.rolledBack || creator.calls != 0 || cache.calls != 0 {
		t.Fatalf("override result=%+v err=%v rollback=%v creates=%d cache=%d", result, err, repo.rolledBack, creator.calls, cache.calls)
	}
	if !errors.As(err, &commitError) || commitError.RowError.Code != "MEMBER_MATCH_CHANGED" {
		t.Fatalf("override commit error = %#v", err)
	}
}

func TestDonationImportCommitAllowsManualAccountWhenServerCannotAutoMatch(t *testing.T) {
	accountUsrSeq := 42
	repo := &donationImportRepositoryStub{candidates: map[string][]model.MemberCandidate{
		"모호|3": {{USRSeq: 21, Name: "모호"}, {USRSeq: 22, Name: "모호"}},
	}}
	creator := &donationImportOrderCreatorStub{sequences: map[string]int64{"모호": 7001}}
	cache := &donationCacheInvalidatorStub{}
	service := NewDonationImportService(repo, creator, cache)
	row := commitRow(3, "모호", 20000, &accountUsrSeq)
	row.ManualOverride = false

	result, err := service.Commit([]model.DonationImportCommitRow{row}, "2026-08-19", 9, "192.0.2.10")
	if err != nil || result == nil || len(result.Rows) != 1 || repo.rolledBack || creator.calls != 1 || cache.calls != 1 {
		t.Fatalf("manual result=%+v err=%v rollback=%v creates=%d cache=%d", result, err, repo.rolledBack, creator.calls, cache.calls)
	}
}

func TestDonationImportCommitRejectsMoreThanMaximumRows(t *testing.T) {
	repo := &donationImportRepositoryStub{}
	service := NewDonationImportService(repo, &donationImportOrderCreatorStub{})
	rows := make([]model.DonationImportCommitRow, donationImportMaxRows+1)
	for index := range rows {
		rows[index] = commitRow(index+3, fmt.Sprintf("기부자%d", index), int64(index+1), nil)
	}

	result, err := service.Commit(rows, "2026-08-19", 9, "192.0.2.10")
	if result != nil || err == nil {
		t.Fatalf("Commit() = %+v, %v; want row limit rejection", result, err)
	}
	var commitError *DonationImportCommitError
	if !errors.As(err, &commitError) || commitError.RowError.Code != "ROW_LIMIT_EXCEEDED" || !strings.Contains(commitError.RowError.Message, "500행") {
		t.Fatalf("commit error = %#v", err)
	}
}

func TestDonationImportCommitRejectsAmountAboveLimit(t *testing.T) {
	service := NewDonationImportService(&donationImportRepositoryStub{}, &donationImportOrderCreatorStub{})
	row := commitRow(3, "고액", donationImportMaxAmount+1, nil)

	result, err := service.Commit([]model.DonationImportCommitRow{row}, "2026-08-19", 9, "192.0.2.10")
	if result != nil || err == nil {
		t.Fatalf("Commit() = %+v, %v; want amount limit rejection", result, err)
	}
	var commitError *DonationImportCommitError
	if !errors.As(err, &commitError) || commitError.RowError.Code != "AMOUNT_LIMIT_EXCEEDED" {
		t.Fatalf("commit error = %#v", err)
	}
}

func TestCreateImportedOrderUsesCanonicalDonationWriter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	adminService := NewAdminDonationService(repository.NewAdminDonationRepository(sqlx.NewDb(db, "sqlmock")), nil)
	accountUsrSeq := 42

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FOR UPDATE`).
		WithArgs(accountUsrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(accountUsrSeq))
	mock.ExpectExec(`INSERT INTO WEO_ORDER \([\s\S]*O_ACCOUNT_USR_SEQ[\s\S]*O_SOURCE`).
		WithArgs(
			accountUsrSeq, accountUsrSeq, "happy_nanum", nil, "cccac1e1b1a20ff31aad795d61129d78adbc2635cc663a01543f1f373e54edde", "2026-08-19",
			"김동문", "01012345678", "11", "총무부", "S",
			int64(100000), int64(0), int64(100000), "completed", "other", nil,
			int64(100000), int64(100000), "FREE", "Y", "Y", 9, "192.0.2.10", 9, "192.0.2.10",
		).
		WillReturnResult(sqlmock.NewResult(7001, 1))
	mock.ExpectCommit()

	tx, err := adminService.repo.DB.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	seq, err := adminService.CreateImportedOrderTx(tx, model.ImportedDonationRow{
		ExcelDonationRow: model.ExcelDonationRow{RowIndex: 2, DonorName: "김동문", DonorCohort: "11", DonorDepartment: "총무부", DonorPhone: "01012345678", Amount: 100000},
		DonationDate:     "2026-08-19",
	}, &accountUsrSeq, 9, "192.0.2.10")
	if err != nil || seq != 7001 {
		t.Fatalf("CreateImportedOrder() = %d, %v", seq, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type donationImportRepositoryStub struct {
	candidates map[string][]model.MemberCandidate
	duplicates map[string]bool
	rolledBack bool
}

func (s *donationImportRepositoryStub) FindMemberCandidatesByNameCohortPhone(name, cohort, _ string) ([]model.MemberCandidate, error) {
	return s.candidates[name+"|"+cohort], nil
}

func (s *donationImportRepositoryStub) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	return s.duplicates[transactionNo] || s.duplicates[compositeKey], nil
}

func (s *donationImportRepositoryStub) FindMemberCandidatesByNameCohortPhoneTx(_ *sqlx.Tx, name, cohort, phone string) ([]model.MemberCandidate, error) {
	return s.FindMemberCandidatesByNameCohortPhone(name, cohort, phone)
}

func (s *donationImportRepositoryStub) ExtRefExistsTx(_ *sqlx.Tx, transactionNo, compositeKey string) (bool, error) {
	return s.ExtRefExists(transactionNo, compositeKey)
}

func (s *donationImportRepositoryStub) RunInTransaction(operation func(*sqlx.Tx) error) error {
	err := operation(nil)
	if err != nil {
		s.rolledBack = true
	}
	return err
}

type donationImportOrderCreatorStub struct {
	sequences map[string]int64
	errors    map[string]error
	calls     int
}

type donationCacheInvalidatorStub struct{ calls int }

func (s *donationCacheInvalidatorStub) InvalidateCache() { s.calls++ }

func (s *donationImportOrderCreatorStub) CreateImportedOrderTx(_ *sqlx.Tx, row model.ImportedDonationRow, _ *int, _ int, _ string) (int64, error) {
	s.calls++
	if err := s.errors[row.DonorName]; err != nil {
		return 0, err
	}
	return s.sequences[row.DonorName], nil
}

func excelWorkbook(t *testing.T, rows [][]interface{}) *bytes.Reader {
	t.Helper()
	workbook := excelize.NewFile()
	sheet := workbook.GetSheetName(0)
	for index, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, index+1)
		if err != nil {
			t.Fatal(err)
		}
		if err := workbook.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		t.Fatal(err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buffer.Bytes())
}

func commitRow(rowIndex int, donorName string, amount int64, accountUsrSeq *int) model.DonationImportCommitRow {
	return model.DonationImportCommitRow{
		ExcelDonationRow: model.ExcelDonationRow{
			RowIndex: rowIndex, DonorName: donorName, DonorCohort: fmt.Sprintf("%d", rowIndex), DonorDepartment: "학과", DonorPhone: "010-1234-5678", Amount: amount,
		},
		AccountUsrSeq: accountUsrSeq,
	}
}
