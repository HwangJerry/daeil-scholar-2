package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/xuri/excelize/v2"
)

// Happy Nanum export header aliases are intentionally centralized here. Update
// these lists when an authoritative export sample becomes available.
var donationImportHeaderCandidates = struct {
	Name          []string
	Phone         []string
	Amount        []string
	DonationDate  []string
	TransactionNo []string
}{
	Name:          []string{"성명", "이름"},
	Phone:         []string{"연락처", "전화번호"},
	Amount:        []string{"입금액", "금액"},
	DonationDate:  []string{"입금일자", "후원일자", "일자"},
	TransactionNo: []string{"거래번호", "승인번호", "고유번호"},
}

var ErrInvalidDonationImportFile = errors.New("invalid donation import file")

type donationImportRepository interface {
	FindMemberCandidatesByNamePhone(name, phone string) ([]model.MemberCandidate, error)
	ExtRefExists(transactionNo, compositeKey string) (bool, error)
}

type donationImportOrderCreator interface {
	CreateImportedOrder(row model.ExcelDonationRow, accountUsrSeq *int, compositeKey string, operSeq int, ip string) (int64, error)
}

type DonationImportService struct {
	repo         donationImportRepository
	orderCreator donationImportOrderCreator
}

func NewDonationImportService(repo donationImportRepository, orderCreator donationImportOrderCreator) *DonationImportService {
	return &DonationImportService{repo: repo, orderCreator: orderCreator}
}

func (s *DonationImportService) ParsePreview(file io.Reader) (*model.DonationImportPreviewResult, error) {
	rows, err := parseExcelDonationRows(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDonationImportFile, err)
	}

	result := &model.DonationImportPreviewResult{Rows: make([]model.DonationImportPreviewRow, 0, len(rows))}
	for _, row := range rows {
		previewRow := model.DonationImportPreviewRow{ExcelDonationRow: row}
		candidates, err := s.repo.FindMemberCandidatesByNamePhone(row.DonorName, row.DonorPhone)
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

		transactionNo, compositeKey := donationImportIdentity(row)
		duplicate, err := s.repo.ExtRefExists(transactionNo, compositeKey)
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

func (s *DonationImportService) Commit(rows []model.DonationImportCommitRow, adminUsrSeq int, ip string) (*model.DonationImportCommitResult, error) {
	result := &model.DonationImportCommitResult{Rows: make([]model.DonationImportCommitRowResult, 0, len(rows))}
	for _, row := range rows {
		rowResult := model.DonationImportCommitRowResult{RowIndex: row.RowIndex}
		normalizedRow, err := normalizeDonationImportCommitRow(row)
		if err != nil {
			rowResult.ErrorMessage = err.Error()
			result.Rows = append(result.Rows, rowResult)
			continue
		}

		transactionNo, compositeKey := donationImportIdentity(normalizedRow.ExcelDonationRow)
		duplicate, err := s.repo.ExtRefExists(transactionNo, compositeKey)
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
			normalizedRow.ExcelDonationRow,
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

func (s *AdminDonationService) CreateImportedOrder(row model.ExcelDonationRow, accountUsrSeq *int, compositeKey string, operSeq int, ip string) (int64, error) {
	var transactionNumber *string
	if row.TransactionNo != "" {
		transactionNumber = &row.TransactionNo
	}
	return s.repo.CreateDonationOrder(model.NormalizedDonationOrder{
		DonationOrderInput: model.DonationOrderInput{
			Source:            "happy_nanum",
			AccountUsrSeq:     accountUsrSeq,
			AccountUsrSeqSet:  true,
			TransactionNumber: transactionNumber,
			DonationDate:      row.DonationDate,
			Donor: model.DonationDonor{
				Name:  row.DonorName,
				Phone: row.DonorPhone,
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

type donationImportColumns struct {
	name          int
	phone         int
	amount        int
	donationDate  int
	transactionNo int
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
	headerRowIndex, columns, err := findDonationImportHeaders(worksheetRows)
	if err != nil {
		return nil, err
	}

	rows := make([]model.ExcelDonationRow, 0, len(worksheetRows)-headerRowIndex-1)
	for index := headerRowIndex + 1; index < len(worksheetRows); index++ {
		worksheetRow := worksheetRows[index]
		if isBlankExcelRow(worksheetRow) {
			continue
		}
		amount, err := parseDonationImportAmount(cellAt(worksheetRow, columns.amount))
		if err != nil {
			return nil, fmt.Errorf("%d행 입금액: %w", index+1, err)
		}
		rows = append(rows, model.ExcelDonationRow{
			RowIndex:      index + 1,
			DonorName:     strings.TrimSpace(cellAt(worksheetRow, columns.name)),
			DonorPhone:    strings.TrimSpace(cellAt(worksheetRow, columns.phone)),
			Amount:        amount,
			DonationDate:  normalizeDonationImportDate(cellAt(worksheetRow, columns.donationDate)),
			TransactionNo: strings.TrimSpace(cellAt(worksheetRow, columns.transactionNo)),
		})
	}
	return rows, nil
}

func findDonationImportHeaders(rows [][]string) (int, donationImportColumns, error) {
	for rowIndex, row := range rows {
		columns := donationImportColumns{name: -1, phone: -1, amount: -1, donationDate: -1, transactionNo: -1}
		for columnIndex, header := range row {
			switch {
			case headerMatches(header, donationImportHeaderCandidates.Name):
				if columns.name == -1 {
					columns.name = columnIndex
				}
			case headerMatches(header, donationImportHeaderCandidates.Phone):
				if columns.phone == -1 {
					columns.phone = columnIndex
				}
			case headerMatches(header, donationImportHeaderCandidates.Amount):
				if columns.amount == -1 {
					columns.amount = columnIndex
				}
			case headerMatches(header, donationImportHeaderCandidates.DonationDate):
				if columns.donationDate == -1 {
					columns.donationDate = columnIndex
				}
			case headerMatches(header, donationImportHeaderCandidates.TransactionNo):
				if columns.transactionNo == -1 {
					columns.transactionNo = columnIndex
				}
			}
		}

		hasRecognizedHeader := columns.name >= 0 || columns.phone >= 0 || columns.amount >= 0 || columns.donationDate >= 0 || columns.transactionNo >= 0
		if !hasRecognizedHeader {
			continue
		}
		missing := make([]string, 0, 3)
		if columns.name < 0 {
			missing = append(missing, "성명/이름")
		}
		if columns.phone < 0 {
			missing = append(missing, "연락처/전화번호")
		}
		if columns.amount < 0 {
			missing = append(missing, "입금액/금액")
		}
		if len(missing) > 0 {
			return 0, donationImportColumns{}, fmt.Errorf("필수 헤더를 찾을 수 없습니다: %s", strings.Join(missing, ", "))
		}
		return rowIndex, columns, nil
	}
	return 0, donationImportColumns{}, errors.New("필수 헤더를 찾을 수 없습니다: 성명/이름, 연락처/전화번호, 입금액/금액")
}

func headerMatches(value string, candidates []string) bool {
	normalizedValue := normalizeDonationImportHeader(value)
	for _, candidate := range candidates {
		if normalizedValue == normalizeDonationImportHeader(candidate) {
			return true
		}
	}
	return false
}

func normalizeDonationImportHeader(value string) string {
	return strings.ToLower(strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, strings.TrimSpace(value)))
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
	row.DonorPhone = model.NormalizePhoneNumber(row.DonorPhone).String()
	row.DonationDate = normalizeDonationImportDate(row.DonationDate)
	row.TransactionNo = strings.TrimSpace(row.TransactionNo)
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
	parsedDate, err := time.Parse("2006-01-02", row.DonationDate)
	if err != nil || parsedDate.Format("2006-01-02") != row.DonationDate {
		return row, errors.New("후원일자가 YYYY-MM-DD 형식의 유효한 날짜가 아닙니다")
	}
	if row.TransactionNo != "" && (len(row.TransactionNo) > 191 || !isASCII(row.TransactionNo)) {
		return row, errors.New("거래번호는 191자 이하의 ASCII 문자열이어야 합니다")
	}
	if row.AccountUsrSeq != nil && *row.AccountUsrSeq <= 0 {
		return row, errors.New("accountUsrSeq는 1 이상이어야 합니다")
	}
	return row, nil
}

func donationImportIdentity(row model.ExcelDonationRow) (string, string) {
	transactionNo := strings.TrimSpace(row.TransactionNo)
	if transactionNo != "" {
		return transactionNo, ""
	}
	identity := fmt.Sprintf(
		"happy_nanum\x00%s\x00%s\x00%s\x00%d",
		normalizeDonationImportDate(row.DonationDate),
		model.NormalizePhoneNumber(row.DonorPhone).String(),
		strings.TrimSpace(row.DonorName),
		row.Amount,
	)
	return "", fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
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
