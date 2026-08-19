package service

import (
	"bytes"
	"errors"
	"fmt"
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

	rows, err := parseExcelDonationRows(workbook)
	if err != nil {
		t.Fatalf("parseExcelDonationRows() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.RowIndex != 3 || row.DonorName != "김동문" || row.DonorCohort != "12" || row.DonorDepartment != "총무부" || row.DonorPhone != "01012345678" || row.Amount != 100000 {
		t.Fatalf("row = %+v", row)
	}
}

func TestParseExcelDonationRowsSkipsRowsWithoutNameOrPhone(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"제목"},
		{"헤더"},
		{"", "", "1", "부서", "010-1111-1111", "10000"},
		{"", "이름", "1", "부서", "", "10000"},
	})

	rows, err := parseExcelDonationRows(workbook)
	if err != nil || len(rows) != 0 {
		t.Fatalf("parseExcelDonationRows() = %+v, %v; want empty rows", rows, err)
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
	repo.duplicates[donationImportIdentity(duplicateRow)] = true

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

func TestDonationImportCommitRechecksDuplicatesAndRecordsPerRowResults(t *testing.T) {
	repo := &donationImportRepositoryStub{duplicates: map[string]bool{}}
	creator := &donationImportOrderCreatorStub{
		sequences: map[string]int64{"성공": 7001},
		errors: map[string]error{
			"경합": repository.ErrDonationOrderConflict,
			"실패": errors.New("insert failed"),
		},
	}
	service := NewDonationImportService(repo, creator)
	accountUsrSeq := 42
	rows := []model.DonationImportCommitRow{
		commitRow(2, "중복", 10000, &accountUsrSeq),
		commitRow(3, "성공", 20000, &accountUsrSeq),
		commitRow(4, "경합", 30000, nil),
		commitRow(5, "실패", 40000, nil),
		commitRow(6, "잘못", 0, nil),
	}
	repo.duplicates[donationImportIdentity(model.ImportedDonationRow{ExcelDonationRow: rows[0].ExcelDonationRow, DonationDate: "2026-08-19"})] = true

	result, err := service.Commit(rows, "2026-08-19", 9, "192.0.2.10")
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if len(result.Rows) != 5 {
		t.Fatalf("result rows = %d, want 5", len(result.Rows))
	}
	if result.Rows[0].Success || result.Rows[0].ErrorMessage != "이미 반영된 기부 거래입니다." {
		t.Fatalf("duplicate result = %+v", result.Rows[0])
	}
	if !result.Rows[1].Success || result.Rows[1].OrderSeq == nil || *result.Rows[1].OrderSeq != 7001 {
		t.Fatalf("success result = %+v", result.Rows[1])
	}
	if result.Rows[2].Success || result.Rows[2].ErrorMessage != "이미 반영된 기부 거래입니다." {
		t.Fatalf("race result = %+v", result.Rows[2])
	}
	if result.Rows[3].Success || result.Rows[3].ErrorMessage != "insert failed" {
		t.Fatalf("failure result = %+v", result.Rows[3])
	}
	if result.Rows[4].Success || result.Rows[4].ErrorMessage != "입금액은 0보다 커야 합니다" {
		t.Fatalf("validation result = %+v", result.Rows[4])
	}
	if creator.calls != 3 {
		t.Fatalf("creator calls = %d, want 3 (duplicate and invalid skipped)", creator.calls)
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
			accountUsrSeq, accountUsrSeq, "happy_nanum", nil, "", "2026-08-19",
			"김동문", "01012345678", "11", "총무부", "S",
			int64(100000), int64(0), int64(100000), "completed", "other", nil,
			int64(100000), int64(100000), "FREE", "Y", "Y", 9, "192.0.2.10", 9, "192.0.2.10",
		).
		WillReturnResult(sqlmock.NewResult(7001, 1))
	mock.ExpectCommit()

	seq, err := adminService.CreateImportedOrder(model.ImportedDonationRow{
		ExcelDonationRow: model.ExcelDonationRow{RowIndex: 2, DonorName: "김동문", DonorCohort: "11", DonorDepartment: "총무부", DonorPhone: "01012345678", Amount: 100000},
		DonationDate:     "2026-08-19",
	}, &accountUsrSeq, "", 9, "192.0.2.10")
	if err != nil || seq != 7001 {
		t.Fatalf("CreateImportedOrder() = %d, %v", seq, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type donationImportRepositoryStub struct {
	candidates map[string][]model.MemberCandidate
	duplicates map[string]bool
}

func (s *donationImportRepositoryStub) FindMemberCandidatesByNameCohortPhone(name, cohort, _ string) ([]model.MemberCandidate, error) {
	return s.candidates[name+"|"+cohort], nil
}

func (s *donationImportRepositoryStub) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	return s.duplicates[transactionNo] || s.duplicates[compositeKey], nil
}

type donationImportOrderCreatorStub struct {
	sequences map[string]int64
	errors    map[string]error
	calls     int
}

func (s *donationImportOrderCreatorStub) CreateImportedOrder(row model.ImportedDonationRow, _ *int, _ string, _ int, _ string) (int64, error) {
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
			RowIndex: rowIndex, DonorName: donorName, DonorCohort: fmt.Sprintf("%d", rowIndex), DonorPhone: "010-1234-5678", Amount: amount,
		},
		AccountUsrSeq: accountUsrSeq,
	}
}
