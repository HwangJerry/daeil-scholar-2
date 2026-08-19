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

func TestParseExcelDonationRowsAcceptsHeaderAliases(t *testing.T) {
	tests := []struct {
		name    string
		headers []interface{}
	}{
		{name: "primary aliases", headers: []interface{}{"성명", "연락처", "입금액", "입금일자", "거래번호"}},
		{name: "secondary aliases", headers: []interface{}{"이름", "전화번호", "금액", "후원일자", "승인번호"}},
		{name: "spaces ignored and optional transaction absent", headers: []interface{}{"성 명", "전 화 번 호", "입 금 액", "일 자"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workbook := excelWorkbook(t, [][]interface{}{
				{"해피나눔 후원 내역"},
				test.headers,
				{"김동문", "010-1234-5678", "100,000원", "2026.08.19", "TX-1"},
			})
			rows, err := parseExcelDonationRows(workbook)
			if err != nil {
				t.Fatalf("parseExcelDonationRows() error = %v", err)
			}
			if len(rows) != 1 || rows[0].RowIndex != 3 || rows[0].Amount != 100000 || rows[0].DonationDate != "2026-08-19" {
				t.Fatalf("rows = %+v", rows)
			}
			if len(test.headers) == 4 && rows[0].TransactionNo != "" {
				t.Fatalf("transactionNo = %q, want empty", rows[0].TransactionNo)
			}
		})
	}
}

func TestParseExcelDonationRowsReportsMissingRequiredHeaders(t *testing.T) {
	workbook := excelWorkbook(t, [][]interface{}{
		{"성명", "입금액", "후원일자"},
		{"김동문", "100000", "2026-08-19"},
	})

	_, err := parseExcelDonationRows(workbook)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("연락처/전화번호")) {
		t.Fatalf("error = %v, want missing phone header message", err)
	}
}

func TestDonationImportPreviewClassifiesEveryStatus(t *testing.T) {
	repo := &donationImportRepositoryStub{
		candidates: map[string][]model.MemberCandidate{
			"매칭": {{USRSeq: 11, Name: "매칭"}},
			"모호": {{USRSeq: 21, Name: "모호"}, {USRSeq: 22, Name: "모호"}},
			"중복": {{USRSeq: 31, Name: "중복"}},
		},
		duplicates: map[string]bool{"DUP-1": true},
	}
	service := NewDonationImportService(repo, &donationImportOrderCreatorStub{})
	workbook := excelWorkbook(t, [][]interface{}{
		{"성명", "연락처", "입금액", "입금일자", "거래번호"},
		{"매칭", "01011111111", "10000", "2026-08-19", "MATCH-1"},
		{"모호", "01022222222", "20000", "2026-08-19", "AMB-1"},
		{"없음", "01033333333", "30000", "2026-08-19", "NONE-1"},
		{"중복", "01044444444", "40000", "2026-08-19", "DUP-1"},
	})

	result, err := service.ParsePreview(workbook)
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
}

func TestDonationImportCommitRechecksDuplicatesAndRecordsPerRowResults(t *testing.T) {
	repo := &donationImportRepositoryStub{duplicates: map[string]bool{"DUP-1": true}}
	creator := &donationImportOrderCreatorStub{
		sequences: map[string]int64{"OK-1": 7001},
		errors: map[string]error{
			"RACE-1": repository.ErrDonationOrderConflict,
			"FAIL-1": errors.New("insert failed"),
		},
	}
	service := NewDonationImportService(repo, creator)
	accountUsrSeq := 42
	rows := []model.DonationImportCommitRow{
		commitRow(2, "DUP-1", 10000, &accountUsrSeq),
		commitRow(3, "OK-1", 20000, &accountUsrSeq),
		commitRow(4, "RACE-1", 30000, nil),
		commitRow(5, "FAIL-1", 40000, nil),
		commitRow(6, "INVALID-1", 0, nil),
	}

	result, err := service.Commit(rows, 9, "192.0.2.10")
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
			accountUsrSeq, accountUsrSeq, "happy_nanum", "TX-1", nil, "2026-08-19",
			"김동문", "01012345678", "", "", "S",
			int64(100000), int64(0), int64(100000), "completed", "other", nil,
			int64(100000), int64(100000), "FREE", "Y", "Y", 9, "192.0.2.10", 9, "192.0.2.10",
		).
		WillReturnResult(sqlmock.NewResult(7001, 1))
	mock.ExpectCommit()

	seq, err := adminService.CreateImportedOrder(model.ExcelDonationRow{
		RowIndex: 2, DonorName: "김동문", DonorPhone: "01012345678", Amount: 100000,
		DonationDate: "2026-08-19", TransactionNo: "TX-1",
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

func (s *donationImportRepositoryStub) FindMemberCandidatesByNamePhone(name, _ string) ([]model.MemberCandidate, error) {
	return s.candidates[name], nil
}

func (s *donationImportRepositoryStub) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	return s.duplicates[transactionNo] || s.duplicates[compositeKey], nil
}

type donationImportOrderCreatorStub struct {
	sequences map[string]int64
	errors    map[string]error
	calls     int
}

func (s *donationImportOrderCreatorStub) CreateImportedOrder(row model.ExcelDonationRow, _ *int, _ string, _ int, _ string) (int64, error) {
	s.calls++
	if err := s.errors[row.TransactionNo]; err != nil {
		return 0, err
	}
	return s.sequences[row.TransactionNo], nil
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

func commitRow(rowIndex int, transactionNo string, amount int64, accountUsrSeq *int) model.DonationImportCommitRow {
	return model.DonationImportCommitRow{
		ExcelDonationRow: model.ExcelDonationRow{
			RowIndex: rowIndex, DonorName: fmt.Sprintf("기부자%d", rowIndex), DonorPhone: "010-1234-5678",
			Amount: amount, DonationDate: "2026-08-19", TransactionNo: transactionNo,
		},
		AccountUsrSeq: accountUsrSeq,
	}
}
