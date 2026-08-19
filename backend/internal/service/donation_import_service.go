package service

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"
)

const (
	donationImportHeaderRowIndex    = 1
	donationImportFirstDataRowIndex = 2
	donationImportNameColumn        = 1
	donationImportCohortColumn      = 2
	donationImportDepartmentColumn  = 3
	donationImportPhoneColumn       = 4
	donationImportAmountColumn      = 5
	donationImportMaxRows           = 500
	donationImportMaxAmount         = int64(1_000_000_000_000)
	donationImportUnzipSizeLimit    = 50 << 20
)

type donationImportHeaderRule struct {
	columnIndex int
	columnName  string
	wantLabel   string
	keywords    []string
}

var donationImportHeaderRules = []donationImportHeaderRule{
	{columnIndex: donationImportNameColumn, columnName: "B", wantLabel: "이름", keywords: []string{"이름", "성명", "회원명"}},
	{columnIndex: donationImportCohortColumn, columnName: "C", wantLabel: "기수", keywords: []string{"기수", "회차"}},
	{columnIndex: donationImportDepartmentColumn, columnName: "D", wantLabel: "학과 또는 부서", keywords: []string{"과", "부서", "소속"}},
	{columnIndex: donationImportPhoneColumn, columnName: "E", wantLabel: "전화 또는 연락처", keywords: []string{"전화", "연락처"}},
}

var (
	ErrInvalidDonationImportFile = errors.New("invalid donation import file")
	ErrInvalidDonationDate       = errors.New("invalid donation date")
	donationImportPhonePattern   = regexp.MustCompile(`^010(?:-[0-9]{4}-[0-9]{4}|[0-9]{8})$`)
)

type DonationImportFileValidationError struct {
	Errors []model.DonationImportRowError
}

func (e *DonationImportFileValidationError) Error() string {
	return fmt.Sprintf("기부 엑셀에 %d개의 오류가 있습니다.", len(e.Errors))
}

type DonationImportCommitError struct{ RowError model.DonationImportRowError }

func (e *DonationImportCommitError) Error() string {
	if e.RowError.RowIndex <= 0 {
		return e.RowError.Message
	}
	return fmt.Sprintf("%d행 %s", e.RowError.RowIndex, e.RowError.Message)
}

type donationImportRepository interface {
	FindMemberCandidatesByNameCohortPhone(name, cohort, phone string) ([]model.MemberCandidate, error)
	FindMemberCandidatesByNameCohortPhoneTx(tx *sqlx.Tx, name, cohort, phone string) ([]model.MemberCandidate, error)
	ExtRefExists(transactionNo, compositeKey string) (bool, error)
	ExtRefExistsTx(tx *sqlx.Tx, transactionNo, compositeKey string) (bool, error)
	RunInTransaction(operation func(*sqlx.Tx) error) error
}

type donationImportOrderCreator interface {
	CreateImportedOrderTx(tx *sqlx.Tx, row model.ImportedDonationRow, accountUsrSeq *int, operSeq int, ip string) (int64, error)
}
type donationCacheInvalidator interface{ InvalidateCache() }

type DonationImportService struct {
	repo             donationImportRepository
	orderCreator     donationImportOrderCreator
	cacheInvalidator donationCacheInvalidator
}

func NewDonationImportService(repo donationImportRepository, orderCreator donationImportOrderCreator, invalidators ...donationCacheInvalidator) *DonationImportService {
	s := &DonationImportService{repo: repo, orderCreator: orderCreator}
	if len(invalidators) > 0 {
		s.cacheInvalidator = invalidators[0]
	}
	return s
}

func (s *DonationImportService) ParsePreview(file io.Reader, donationDate string) (*model.DonationImportPreviewResult, error) {
	normalizedDate, err := validateDonationImportDate(donationDate)
	if err != nil {
		return nil, err
	}
	rows, validationErrors, err := parseExcelDonationRows(file, normalizedDate)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDonationImportFile, err)
	}
	if len(validationErrors) > 0 {
		return nil, &DonationImportFileValidationError{Errors: validationErrors}
	}

	result := &model.DonationImportPreviewResult{Rows: make([]model.DonationImportPreviewRow, 0, len(rows))}
	for _, row := range rows {
		previewRow := model.DonationImportPreviewRow{ExcelDonationRow: row, DonationDate: normalizedDate}
		candidates, err := s.repo.FindMemberCandidatesByNameCohortPhone(row.DonorName, row.DonorCohort, row.DonorPhone)
		if err != nil {
			return nil, fmt.Errorf("%d행 회원 조회: %w", row.RowIndex, err)
		}
		switch len(candidates) {
		case 0:
			previewRow.Status, previewRow.Note = model.DonationImportRowUnmatched, "일치하는 활성 회원 없음"
		case 1:
			previewRow.Status = model.DonationImportRowMatched
			previewRow.MatchedUsrSeq, previewRow.MatchedName = intPointer(candidates[0].USRSeq), candidates[0].Name
		default:
			previewRow.Status, previewRow.Note = model.DonationImportRowAmbiguous, fmt.Sprintf("동명이인 %d명 발견", len(candidates))
		}
		key, err := importedDonationCompositeKey(model.ImportedDonationRow{ExcelDonationRow: row, DonationDate: normalizedDate})
		if err != nil {
			return nil, err
		}
		duplicate, err := s.repo.ExtRefExists("", key)
		if err != nil {
			return nil, fmt.Errorf("%d행 중복 조회: %w", row.RowIndex, err)
		}
		if duplicate {
			previewRow.Status, previewRow.Note = model.DonationImportRowDuplicate, "이미 반영된 기부 거래"
		}
		result.Rows = append(result.Rows, previewRow)
		incrementDonationImportCount(result, previewRow.Status)
	}
	return result, nil
}

func (s *DonationImportService) Commit(rows []model.DonationImportCommitRow, donationDate string, adminUsrSeq int, ip string) (*model.DonationImportCommitResult, error) {
	normalizedDate, err := validateDonationImportDate(donationDate)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, newCommitError(0, "rows", "ROWS_REQUIRED", "반영할 행이 없습니다.")
	}
	if len(rows) > donationImportMaxRows {
		return nil, newCommitError(0, "rows", "ROW_LIMIT_EXCEEDED", "한 번에 최대 500행까지 반영할 수 있습니다.")
	}

	normalizedRows := make([]model.DonationImportCommitRow, 0, len(rows))
	seenKeys := make(map[string]int, len(rows))
	for _, row := range rows {
		normalizedRow, rowErr := normalizeDonationImportCommitRow(row)
		if rowErr != nil {
			return nil, &DonationImportCommitError{RowError: *rowErr}
		}
		key, normalizeErr := importedDonationCompositeKey(model.ImportedDonationRow{ExcelDonationRow: normalizedRow.ExcelDonationRow, DonationDate: normalizedDate})
		if normalizeErr != nil {
			return nil, newCommitError(row.RowIndex, "row", "INVALID_ROW", "행 값이 올바르지 않습니다.")
		}
		if first, duplicate := seenKeys[key]; duplicate {
			return nil, newCommitError(row.RowIndex, "compositeKey", "DUPLICATE_IN_FILE", fmt.Sprintf("%d행과 동일한 기부 내역입니다.", first))
		}
		seenKeys[key] = row.RowIndex
		normalizedRows = append(normalizedRows, normalizedRow)
	}

	result := &model.DonationImportCommitResult{Rows: make([]model.DonationImportCommitRowResult, 0, len(rows))}
	err = s.repo.RunInTransaction(func(tx *sqlx.Tx) error {
		for _, row := range normalizedRows {
			importedRow := model.ImportedDonationRow{ExcelDonationRow: row.ExcelDonationRow, DonationDate: normalizedDate}
			key, _ := importedDonationCompositeKey(importedRow)
			duplicate, lookupErr := s.repo.ExtRefExistsTx(tx, "", key)
			if lookupErr != nil {
				return internalCommitFailure(row.RowIndex, "중복 확인 중 오류가 발생했습니다.", lookupErr)
			}
			if duplicate {
				return newCommitError(row.RowIndex, "compositeKey", "ALREADY_IMPORTED", "이미 반영된 기부 거래입니다.")
			}

			candidates, matchErr := s.repo.FindMemberCandidatesByNameCohortPhoneTx(tx, row.DonorName, row.DonorCohort, row.DonorPhone)
			if matchErr != nil {
				return internalCommitFailure(row.RowIndex, "회원 재확인 중 오류가 발생했습니다.", matchErr)
			}
			serverCanAutoMatch := len(candidates) == 1
			clientConfirmedAutoMatch := serverCanAutoMatch && row.AccountUsrSeq != nil && candidates[0].USRSeq == *row.AccountUsrSeq
			if serverCanAutoMatch && !clientConfirmedAutoMatch {
				return newCommitError(row.RowIndex, "accountUsrSeq", "MEMBER_MATCH_CHANGED", "미리보기 이후 회원 매칭 정보가 변경되었습니다. 다시 미리보기해 주세요.")
			}

			orderSeq, createErr := s.orderCreator.CreateImportedOrderTx(tx, importedRow, row.AccountUsrSeq, adminUsrSeq, ip)
			if createErr != nil {
				if errors.Is(createErr, repository.ErrDonationOrderConflict) {
					return newCommitError(row.RowIndex, "compositeKey", "ALREADY_IMPORTED", "이미 반영된 기부 거래입니다.")
				}
				if errors.Is(createErr, repository.ErrDonationAccountNotFound) {
					return newCommitError(row.RowIndex, "accountUsrSeq", "ACCOUNT_NOT_FOUND", "연결할 활성 회원을 찾을 수 없습니다.")
				}
				return internalCommitFailure(row.RowIndex, "저장 중 오류가 발생했습니다.", createErr)
			}
			result.Rows = append(result.Rows, model.DonationImportCommitRowResult{RowIndex: row.RowIndex, Success: true, OrderSeq: int64Pointer(orderSeq)})
		}
		return nil
	})
	if err != nil {
		var failure *donationImportInternalFailure
		if errors.As(err, &failure) {
			log.Error().Err(failure.cause).Int("rowIndex", failure.public.RowError.RowIndex).Msg("donation import commit failed")
			return nil, failure.public
		}
		var publicError *DonationImportCommitError
		if errors.As(err, &publicError) {
			return nil, publicError
		}
		log.Error().Err(err).Msg("donation import transaction failed")
		return nil, newCommitError(0, "rows", "COMMIT_FAILED", "저장 중 오류가 발생했습니다.")
	}
	if s.cacheInvalidator != nil {
		s.cacheInvalidator.InvalidateCache()
	}
	return result, nil
}

type donationImportInternalFailure struct {
	public *DonationImportCommitError
	cause  error
}

func (e *donationImportInternalFailure) Error() string { return e.public.Error() }
func internalCommitFailure(rowIndex int, message string, cause error) error {
	return &donationImportInternalFailure{public: newCommitError(rowIndex, "row", "STORAGE_ERROR", message), cause: cause}
}
func newCommitError(rowIndex int, field, code, message string) *DonationImportCommitError {
	return &DonationImportCommitError{RowError: model.DonationImportRowError{RowIndex: rowIndex, Field: field, Code: code, Message: message}}
}

func parseExcelDonationRows(reader io.Reader, donationDate string) ([]model.ExcelDonationRow, []model.DonationImportRowError, error) {
	workbook, err := excelize.OpenReader(reader, excelize.Options{UnzipSizeLimit: donationImportUnzipSizeLimit})
	if err != nil {
		return nil, nil, fmt.Errorf("엑셀 파일을 열 수 없습니다: %w", err)
	}
	defer workbook.Close()
	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errors.New("엑셀 파일에 시트가 없습니다")
	}
	if len(sheets) != 1 {
		return nil, nil, errors.New("엑셀 파일에는 업로드 시트가 하나만 있어야 합니다")
	}
	worksheetRows, err := workbook.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("엑셀 행을 읽을 수 없습니다: %w", err)
	}
	if err := validateDonationImportHeaders(worksheetRows); err != nil {
		return nil, nil, err
	}

	rows := make([]model.ExcelDonationRow, 0, min(donationImportMaxRows, max(0, len(worksheetRows)-donationImportFirstDataRowIndex)))
	validationErrors := make([]model.DonationImportRowError, 0)
	seenKeys := make(map[string]int)
	dataRowCount := 0
	for index := donationImportFirstDataRowIndex; index < len(worksheetRows); index++ {
		worksheetRow := worksheetRows[index]
		if isBlankExcelRow(worksheetRow) {
			continue
		}
		dataRowCount++
		rowIndex := index + 1
		if dataRowCount > donationImportMaxRows {
			validationErrors = append(validationErrors, model.DonationImportRowError{RowIndex: rowIndex, Field: "rows", Code: "ROW_LIMIT_EXCEEDED", Message: "데이터 행은 최대 500개까지 허용됩니다."})
			break
		}

		name := strings.TrimSpace(cellAt(worksheetRow, donationImportNameColumn))
		cohort := strings.TrimSpace(cellAt(worksheetRow, donationImportCohortColumn))
		department := strings.TrimSpace(cellAt(worksheetRow, donationImportDepartmentColumn))
		if department == "0" {
			department = ""
		}
		phone := strings.TrimSpace(cellAt(worksheetRow, donationImportPhoneColumn))
		amountValue := cellAt(worksheetRow, donationImportAmountColumn)
		rowErrors := validateExcelDonationFields(rowIndex, name, cohort, department, phone, amountValue)
		validationErrors = append(validationErrors, rowErrors...)
		if len(rowErrors) > 0 {
			continue
		}
		amount, _ := parseDonationImportAmount(amountValue)
		row := model.ExcelDonationRow{RowIndex: rowIndex, DonorName: name, DonorCohort: cohort, DonorDepartment: department, DonorPhone: strings.ReplaceAll(phone, "-", ""), Amount: amount}
		key, normalizeErr := importedDonationCompositeKey(model.ImportedDonationRow{ExcelDonationRow: row, DonationDate: donationDate})
		if normalizeErr != nil {
			validationErrors = append(validationErrors, model.DonationImportRowError{RowIndex: rowIndex, Field: "row", Code: "INVALID_ROW", Message: "행 값이 올바르지 않습니다."})
			continue
		}
		if first, duplicate := seenKeys[key]; duplicate {
			validationErrors = append(validationErrors, model.DonationImportRowError{RowIndex: rowIndex, Field: "compositeKey", Code: "DUPLICATE_IN_FILE", Message: fmt.Sprintf("%d행과 동일한 기부 내역입니다.", first)})
			continue
		}
		seenKeys[key] = rowIndex
		rows = append(rows, row)
	}
	return rows, validationErrors, nil
}

func validateDonationImportHeaders(worksheetRows [][]string) error {
	var headerRow []string
	if len(worksheetRows) > donationImportHeaderRowIndex {
		headerRow = worksheetRows[donationImportHeaderRowIndex]
	}
	for _, rule := range donationImportHeaderRules {
		header := strings.TrimSpace(cellAt(headerRow, rule.columnIndex))
		matches := false
		for _, keyword := range rule.keywords {
			if strings.Contains(header, keyword) {
				matches = true
				break
			}
		}
		if !matches {
			return fmt.Errorf("2행 %s열은 '%s' 헤더가 아닙니다", rule.columnName, rule.wantLabel)
		}
	}
	return nil
}

func validateExcelDonationFields(rowIndex int, name, cohort, department, phone, amount string) []model.DonationImportRowError {
	found := make([]model.DonationImportRowError, 0)
	addRequired := func(field, label, value string) {
		if value == "" {
			found = append(found, model.DonationImportRowError{RowIndex: rowIndex, Field: field, Code: "REQUIRED", Message: label + " 값이 필요합니다."})
		}
	}
	addRequired("donorName", "이름", name)
	addRequired("donorCohort", "기수", cohort)
	addRequired("donorDepartment", "학과", department)
	addRequired("donorPhone", "연락처", phone)
	if phone != "" && !donationImportPhonePattern.MatchString(phone) {
		found = append(found, model.DonationImportRowError{RowIndex: rowIndex, Field: "donorPhone", Code: "INVALID_PHONE", Message: "연락처는 010-0000-0000 또는 01000000000 형식이어야 합니다."})
	}
	parsedAmount, amountErr := parseDonationImportAmount(amount)
	if amountErr != nil || parsedAmount <= 0 {
		code := "INVALID_AMOUNT"
		if strings.TrimSpace(amount) == "" {
			code = "REQUIRED"
		}
		found = append(found, model.DonationImportRowError{RowIndex: rowIndex, Field: "amount", Code: code, Message: "금액은 0보다 큰 정수여야 합니다."})
	} else if parsedAmount > donationImportMaxAmount {
		found = append(found, model.DonationImportRowError{RowIndex: rowIndex, Field: "amount", Code: "AMOUNT_LIMIT_EXCEEDED", Message: "금액은 1조원 이하여야 합니다."})
	}
	return found
}

func parseDonationImportAmount(value string) (int64, error) {
	normalized := strings.NewReplacer(",", "", " ", "", "₩", "", "원", "").Replace(strings.TrimSpace(value))
	if normalized == "" {
		return 0, errors.New("값이 비어 있습니다")
	}
	amount, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q은(는) 정수가 아닙니다", value)
	}
	return amount, nil
}

func normalizeDonationImportDate(value string) string {
	value = strings.TrimSpace(value)
	formats := []string{"2006-01-02", "2006.01.02", "2006/01/02", "20060102", "2006년1월2일", "2006년 1월 2일", "2006-01-02 15:04:05"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return value
}

func normalizeDonationImportCommitRow(row model.DonationImportCommitRow) (model.DonationImportCommitRow, *model.DonationImportRowError) {
	row.DonorName, row.DonorCohort, row.DonorDepartment = strings.TrimSpace(row.DonorName), strings.TrimSpace(row.DonorCohort), strings.TrimSpace(row.DonorDepartment)
	rawPhone := strings.TrimSpace(row.DonorPhone)
	if row.RowIndex <= 0 {
		return row, rowError(row.RowIndex, "rowIndex", "INVALID_ROW_INDEX", "rowIndex는 1 이상이어야 합니다.")
	}
	if row.DonorName == "" {
		return row, rowError(row.RowIndex, "donorName", "REQUIRED", "이름 값이 필요합니다.")
	}
	if row.DonorCohort == "" {
		return row, rowError(row.RowIndex, "donorCohort", "REQUIRED", "기수 값이 필요합니다.")
	}
	if row.DonorDepartment == "" {
		return row, rowError(row.RowIndex, "donorDepartment", "REQUIRED", "학과 값이 필요합니다.")
	}
	if !donationImportPhonePattern.MatchString(rawPhone) {
		return row, rowError(row.RowIndex, "donorPhone", "INVALID_PHONE", "연락처는 010-0000-0000 또는 01000000000 형식이어야 합니다.")
	}
	row.DonorPhone = strings.ReplaceAll(rawPhone, "-", "")
	if row.Amount <= 0 {
		return row, rowError(row.RowIndex, "amount", "INVALID_AMOUNT", "금액은 0보다 큰 정수여야 합니다.")
	}
	if row.Amount > donationImportMaxAmount {
		return row, rowError(row.RowIndex, "amount", "AMOUNT_LIMIT_EXCEEDED", "금액은 1조원 이하여야 합니다.")
	}
	if row.AccountUsrSeq != nil && *row.AccountUsrSeq <= 0 {
		return row, rowError(row.RowIndex, "accountUsrSeq", "INVALID_ACCOUNT", "accountUsrSeq는 1 이상이어야 합니다.")
	}
	return row, nil
}

func rowError(rowIndex int, field, code, message string) *model.DonationImportRowError {
	return &model.DonationImportRowError{RowIndex: rowIndex, Field: field, Code: code, Message: message}
}
func importedDonationCompositeKey(row model.ImportedDonationRow) (string, error) {
	normalized, err := normalizeImportedDonationOrder(row, nil)
	if err != nil {
		return "", err
	}
	return normalized.CompositeKey, nil
}
func validateDonationImportDate(value string) (string, error) {
	normalized := normalizeDonationImportDate(value)
	parsed, err := time.Parse("2006-01-02", normalized)
	if err != nil || parsed.Format("2006-01-02") != normalized {
		return "", fmt.Errorf("%w: 기부 반영일자는 YYYY-MM-DD 형식의 유효한 날짜여야 합니다", ErrInvalidDonationDate)
	}
	return normalized, nil
}
func incrementDonationImportCount(result *model.DonationImportPreviewResult, status model.DonationImportRowStatus) {
	switch status {
	case model.DonationImportRowMatched:
		result.MatchedCount++
	case model.DonationImportRowAmbiguous:
		result.AmbiguousCount++
	case model.DonationImportRowUnmatched:
		result.UnmatchedCount++
	case model.DonationImportRowDuplicate:
		result.DuplicateCount++
	}
}
func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}
func isBlankExcelRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
func intPointer(value int) *int       { return &value }
func int64Pointer(value int64) *int64 { return &value }
