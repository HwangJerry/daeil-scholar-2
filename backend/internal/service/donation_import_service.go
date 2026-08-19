package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/xuri/excelize/v2"
)

const (
	donationImportFirstDataRowIndex = 2 // zero-based: Excel row 3
	donationImportNameColumn        = 1 // B
	donationImportCohortColumn      = 2 // C
	donationImportDepartmentColumn  = 3 // D
	donationImportPhoneColumn       = 4 // E
	donationImportAmountColumn      = 5 // F
)

var ErrInvalidDonationImportFile = errors.New("invalid donation import file")
var ErrInvalidDonationDate = errors.New("invalid donation date")

type donationImportRepository interface {
	FindMemberCandidatesByNameCohortPhone(name, cohort, phone string) ([]model.MemberCandidate, error)
	ExtRefExists(transactionNo, compositeKey string) (bool, error)
}

type donationImportOrderCreator interface {
	CreateImportedOrder(row model.ImportedDonationRow, accountUsrSeq *int, compositeKey string, operSeq int, ip string) (int64, error)
}

type DonationImportService struct {
	repo         donationImportRepository
	orderCreator donationImportOrderCreator
}

func NewDonationImportService(repo donationImportRepository, orderCreator donationImportOrderCreator) *DonationImportService {
	return &DonationImportService{repo: repo, orderCreator: orderCreator}
}

func (s *DonationImportService) ParsePreview(file io.Reader, donationDate string) (*model.DonationImportPreviewResult, error) {
	normalizedDonationDate, err := validateDonationImportDate(donationDate)
	if err != nil {
		return nil, err
	}
	rows, err := parseExcelDonationRows(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDonationImportFile, err)
	}

	result := &model.DonationImportPreviewResult{Rows: make([]model.DonationImportPreviewRow, 0, len(rows))}
	for _, row := range rows {
		previewRow := model.DonationImportPreviewRow{ExcelDonationRow: row, DonationDate: normalizedDonationDate}
		candidates, err := s.repo.FindMemberCandidatesByNameCohortPhone(row.DonorName, row.DonorCohort, row.DonorPhone)
		if err != nil {
			return nil, fmt.Errorf("%d행 회원 조회: %w", row.RowIndex, err)
		}
		switch len(candidates) {
		case 0:
			previewRow.Status = model.DonationImportRowUnmatched
			previewRow.Note = "일치하는 활성 회원 없음"
		case 1:
			previewRow.Status = model.DonationImportRowMatched
			previewRow.MatchedUsrSeq = intPointer(candidates[0].USRSeq)
			previewRow.MatchedName = candidates[0].Name
		default:
			previewRow.Status = model.DonationImportRowAmbiguous
			previewRow.Note = fmt.Sprintf("동명이인 %d명 발견", len(candidates))
		}

		compositeKey := donationImportIdentity(model.ImportedDonationRow{ExcelDonationRow: row, DonationDate: normalizedDonationDate})
		duplicate, err := s.repo.ExtRefExists("", compositeKey)
		if err != nil {
			return nil, fmt.Errorf("%d행 중복 조회: %w", row.RowIndex, err)
		}
		if duplicate {
			previewRow.Status = model.DonationImportRowDuplicate
			previewRow.Note = "이미 반영된 기부 거래"
		}

		result.Rows = append(result.Rows, previewRow)
		incrementDonationImportCount(result, previewRow.Status)
	}
	return result, nil
}

func (s *DonationImportService) Commit(rows []model.DonationImportCommitRow, donationDate string, adminUsrSeq int, ip string) (*model.DonationImportCommitResult, error) {
	normalizedDonationDate, err := validateDonationImportDate(donationDate)
	if err != nil {
		return nil, err
	}
	result := &model.DonationImportCommitResult{Rows: make([]model.DonationImportCommitRowResult, 0, len(rows))}
	for _, row := range rows {
		rowResult := model.DonationImportCommitRowResult{RowIndex: row.RowIndex}
		normalizedRow, err := normalizeDonationImportCommitRow(row)
		if err != nil {
			rowResult.ErrorMessage = err.Error()
			result.Rows = append(result.Rows, rowResult)
			continue
		}

		importedRow := model.ImportedDonationRow{ExcelDonationRow: normalizedRow.ExcelDonationRow, DonationDate: normalizedDonationDate}
		compositeKey := donationImportIdentity(importedRow)
		duplicate, err := s.repo.ExtRefExists("", compositeKey)
		if err != nil {
			rowResult.ErrorMessage = fmt.Sprintf("중복 확인 실패: %v", err)
			result.Rows = append(result.Rows, rowResult)
			continue
		}
		if duplicate {
			rowResult.ErrorMessage = "이미 반영된 기부 거래입니다."
			result.Rows = append(result.Rows, rowResult)
			continue
		}

		orderSeq, err := s.orderCreator.CreateImportedOrder(
			importedRow,
			normalizedRow.AccountUsrSeq,
			compositeKey,
			adminUsrSeq,
			ip,
		)
		if err != nil {
			if errors.Is(err, repository.ErrDonationOrderConflict) {
				rowResult.ErrorMessage = "이미 반영된 기부 거래입니다."
			} else {
				rowResult.ErrorMessage = err.Error()
			}
			result.Rows = append(result.Rows, rowResult)
			continue
		}

		rowResult.Success = true
		rowResult.OrderSeq = int64Pointer(orderSeq)
		result.Rows = append(result.Rows, rowResult)
	}
	return result, nil
}

func (s *AdminDonationService) CreateImportedOrder(row model.ImportedDonationRow, accountUsrSeq *int, compositeKey string, operSeq int, ip string) (int64, error) {
	return s.repo.CreateDonationOrder(model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source:            "happy_nanum",
			AccountUsrSeq:     accountUsrSeq,
			AccountUsrSeqSet:  true,
			TransactionNumber: nil,
			DonationDate:      row.DonationDate,
			Donor: model.DonationDonor{
				Name:       row.DonorName,
				Cohort:     row.DonorCohort,
				Department: row.DonorDepartment,
				Phone:      row.DonorPhone,
			},
			DonationType:   "one_time",
			GrossAmount:    row.Amount,
			RefundedAmount: 0,
			Status:         "completed",
			PaymentMethod:  "other",
		},
		NetReceivedAmount: row.Amount,
		CompositeKey:      compositeKey,
		LegacyGate:        "S",
		LegacyStatus:      "Y",
		LegacyPayment:     "Y",
	}, operSeq, ip)
}

func parseExcelDonationRows(reader io.Reader) ([]model.ExcelDonationRow, error) {
	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("엑셀 파일을 열 수 없습니다: %w", err)
	}
	defer workbook.Close()

	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("엑셀 파일에 시트가 없습니다")
	}
	worksheetRows, err := workbook.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("엑셀 행을 읽을 수 없습니다: %w", err)
	}
	rows := make([]model.ExcelDonationRow, 0, max(0, len(worksheetRows)-donationImportFirstDataRowIndex))
	for index := donationImportFirstDataRowIndex; index < len(worksheetRows); index++ {
		worksheetRow := worksheetRows[index]
		if strings.TrimSpace(cellAt(worksheetRow, donationImportNameColumn)) == "" || strings.TrimSpace(cellAt(worksheetRow, donationImportPhoneColumn)) == "" {
			continue
		}
		amount, err := parseDonationImportAmount(cellAt(worksheetRow, donationImportAmountColumn))
		if err != nil {
			return nil, fmt.Errorf("%d행 입금액: %w", index+1, err)
		}
		donorDepartment := strings.TrimSpace(cellAt(worksheetRow, donationImportDepartmentColumn))
		if donorDepartment == "0" {
			donorDepartment = ""
		}
		rows = append(rows, model.ExcelDonationRow{
			RowIndex:        index + 1,
			DonorName:       strings.TrimSpace(cellAt(worksheetRow, donationImportNameColumn)),
			DonorCohort:     strings.TrimSpace(cellAt(worksheetRow, donationImportCohortColumn)),
			DonorDepartment: donorDepartment,
			DonorPhone:      strings.ReplaceAll(strings.TrimSpace(cellAt(worksheetRow, donationImportPhoneColumn)), "-", ""),
			Amount:          amount,
		})
	}
	return rows, nil
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

func normalizeDonationImportCommitRow(row model.DonationImportCommitRow) (model.DonationImportCommitRow, error) {
	row.DonorName = strings.TrimSpace(row.DonorName)
	row.DonorCohort = strings.TrimSpace(row.DonorCohort)
	row.DonorDepartment = strings.TrimSpace(row.DonorDepartment)
	row.DonorPhone = model.NormalizePhoneNumber(row.DonorPhone).String()
	if row.RowIndex <= 0 {
		return row, errors.New("rowIndex는 1 이상이어야 합니다")
	}
	if row.DonorName == "" {
		return row, errors.New("기부자 이름이 비어 있습니다")
	}
	if !model.NormalizePhoneNumber(row.DonorPhone).Valid() {
		return row, errors.New("기부자 전화번호가 올바르지 않습니다")
	}
	if row.Amount <= 0 {
		return row, errors.New("입금액은 0보다 커야 합니다")
	}
	if row.AccountUsrSeq != nil && *row.AccountUsrSeq <= 0 {
		return row, errors.New("accountUsrSeq는 1 이상이어야 합니다")
	}
	return row, nil
}

func donationImportIdentity(row model.ImportedDonationRow) string {
	identity := fmt.Sprintf(
		"happy_nanum\x00%s\x00%s\x00%s\x00%s\x00%s\x00%d",
		normalizeDonationImportDate(row.DonationDate),
		model.NormalizePhoneNumber(row.DonorPhone).String(),
		strings.TrimSpace(row.DonorName),
		strings.TrimSpace(row.DonorCohort),
		strings.TrimSpace(row.DonorDepartment),
		row.Amount,
	)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
}

func validateDonationImportDate(value string) (string, error) {
	normalized := normalizeDonationImportDate(value)
	parsedDate, err := time.Parse("2006-01-02", normalized)
	if err != nil || parsedDate.Format("2006-01-02") != normalized {
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
