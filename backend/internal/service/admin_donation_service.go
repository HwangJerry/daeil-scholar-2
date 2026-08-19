// Admin donation service — business logic for donation config
package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

var (
	ErrInvalidDonationOrder    = errors.New("invalid donation order")
	ErrDonationAccountNotFound = repository.ErrDonationAccountNotFound
)

func NormalizeDonationOrderInput(input model.DonationOrderInput) (normalized model.NormalizedDonationOrder, err error) {
	defer func() {
		if err != nil && !errors.Is(err, ErrInvalidDonationOrder) {
			err = fmt.Errorf("%w: %v", ErrInvalidDonationOrder, err)
		}
	}()
	input.Source = strings.TrimSpace(input.Source)
	input.DonationDate = strings.TrimSpace(input.DonationDate)
	input.Donor.Name = strings.TrimSpace(input.Donor.Name)
	input.Donor.Cohort = strings.TrimSpace(input.Donor.Cohort)
	input.Donor.Department = strings.TrimSpace(input.Donor.Department)
	input.Donor.Phone = strings.TrimSpace(input.Donor.Phone)
	input.DonationType = strings.TrimSpace(input.DonationType)
	input.Status = strings.TrimSpace(input.Status)
	input.PaymentMethod = strings.TrimSpace(input.PaymentMethod)

	switch input.Source {
	case "happy_nanum", "bank_transfer", "other":
	default:
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid source")
	}
	if parsed, err := time.Parse("2006-01-02", input.DonationDate); err != nil || parsed.Format("2006-01-02") != input.DonationDate {
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid donation date")
	}
	if input.Donor.Name == "" || input.Donor.Cohort == "" || input.Donor.Department == "" {
		return model.NormalizedDonationOrder{}, fmt.Errorf("donor fields are required")
	}
	if matched, _ := regexp.MatchString(`^010(?:-[0-9]{4}-[0-9]{4}|[0-9]{8})$`, input.Donor.Phone); !matched {
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid donor phone")
	}
	input.Donor.Phone = strings.ReplaceAll(input.Donor.Phone, "-", "")

	legacyGate := ""
	switch input.DonationType {
	case "recurring":
		legacyGate = "P"
	case "one_time":
		legacyGate = "S"
	case "sponsorship":
		legacyGate = "F"
	default:
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid donation type")
	}
	legacyStatus, legacyPayment := "", ""
	switch input.Status {
	case "scheduled", "pending":
		legacyStatus, legacyPayment = "I", "N"
	case "completed", "partially_refunded":
		legacyStatus, legacyPayment = "Y", "Y"
	case "cancelled", "fully_refunded":
		legacyStatus, legacyPayment = "N", "N"
	default:
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid donation status")
	}
	switch input.PaymentMethod {
	case "card", "bank", "virtual_bank", "mobile", "admin", "other":
	default:
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid payment method")
	}
	if input.TransactionNumber != nil {
		transactionNumber := strings.TrimSpace(*input.TransactionNumber)
		if transactionNumber == "" || len(transactionNumber) > 191 || !isASCII(transactionNumber) {
			return model.NormalizedDonationOrder{}, fmt.Errorf("invalid transaction number")
		}
		input.TransactionNumber = &transactionNumber
	}
	if input.GrossAmount < 0 || input.RefundedAmount < 0 || input.RefundedAmount > input.GrossAmount {
		return model.NormalizedDonationOrder{}, fmt.Errorf("invalid donation amounts")
	}
	switch input.Status {
	case "partially_refunded":
		if input.RefundedAmount == 0 || input.RefundedAmount == input.GrossAmount {
			return model.NormalizedDonationOrder{}, fmt.Errorf("partially_refunded requires a partial refund")
		}
	case "fully_refunded":
		if input.RefundedAmount != input.GrossAmount {
			return model.NormalizedDonationOrder{}, fmt.Errorf("fully_refunded requires a full refund")
		}
	case "scheduled", "pending", "completed", "cancelled":
		if input.RefundedAmount != 0 {
			return model.NormalizedDonationOrder{}, fmt.Errorf("status %s does not allow a refund", input.Status)
		}
	}
	normalized = model.NormalizedDonationOrder{
		DonationOrderInput: input,
		NetReceivedAmount:  input.GrossAmount - input.RefundedAmount,
		LegacyGate:         legacyGate,
		LegacyStatus:       legacyStatus,
		LegacyPayment:      legacyPayment,
	}
	if input.TransactionNumber == nil {
		identity := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%d", input.DonationDate, input.Donor.Phone, input.Donor.Name, input.Donor.Cohort, input.Donor.Department, input.GrossAmount)
		normalized.CompositeKey = fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
	}
	return normalized, nil
}

func isASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 127 {
			return false
		}
	}
	return true
}

type AdminDonationService struct {
	repo         *repository.AdminDonationRepository
	donationRepo *repository.DonationRepository
}

func NewAdminDonationService(
	repo *repository.AdminDonationRepository,
	donationRepo *repository.DonationRepository,
) *AdminDonationService {
	return &AdminDonationService{repo: repo, donationRepo: donationRepo}
}

func (s *AdminDonationService) GetConfig() (*model.DonationConfig, error) {
	return s.donationRepo.GetActiveConfig()
}

func (s *AdminDonationService) UpdateConfig(goal int64, manualAdj int64, manualDonorCnt int, note string, overwrite bool, operSeq int) error {
	ov := "N"
	if overwrite {
		ov = "Y"
	}
	return s.repo.UpdateConfig(goal, manualAdj, manualDonorCnt, note, ov, operSeq)
}

func (s *AdminDonationService) GetHistory(days int) ([]model.DonationSnapshot, error) {
	if days <= 0 {
		days = 30
	}
	return s.repo.GetSnapshotHistory(days)
}

func (s *AdminDonationService) ListOrders(filters model.DonationOrderFilters, page, size int) (*model.DonationOrderPage, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	filters.Name = strings.TrimSpace(filters.Name)
	filters.Phone = strings.ReplaceAll(strings.TrimSpace(filters.Phone), "-", "")
	filters.TransactionNumber = strings.TrimSpace(filters.TransactionNumber)
	filters.Source = strings.TrimSpace(filters.Source)
	filters.Status = strings.TrimSpace(filters.Status)
	filters.DonationType = strings.TrimSpace(filters.DonationType)
	if filters.Source != "" && filters.Source != "happy_nanum" && filters.Source != "bank_transfer" && filters.Source != "other" {
		return nil, fmt.Errorf("%w: invalid donation source filter", ErrInvalidDonationOrder)
	}
	if filters.Status != "" && !isDonationStatus(filters.Status) {
		return nil, fmt.Errorf("%w: invalid donation status filter", ErrInvalidDonationOrder)
	}
	if filters.DonationType != "" && filters.DonationType != "recurring" && filters.DonationType != "one_time" && filters.DonationType != "sponsorship" {
		return nil, fmt.Errorf("%w: invalid donation type filter", ErrInvalidDonationOrder)
	}
	if filters.Phone != "" {
		for _, value := range filters.Phone {
			if value < '0' || value > '9' {
				return nil, fmt.Errorf("%w: invalid donor phone filter", ErrInvalidDonationOrder)
			}
		}
	}
	if filters.TransactionNumber != "" && !isASCII(filters.TransactionNumber) {
		return nil, fmt.Errorf("%w: invalid transaction number filter", ErrInvalidDonationOrder)
	}
	items, total, err := s.repo.ListDonationOrders(filters, page, size)
	if err != nil {
		return nil, err
	}
	return &model.DonationOrderPage{Items: items, Total: total, Page: page, Size: size}, nil
}

func isDonationStatus(status string) bool {
	switch status {
	case "scheduled", "pending", "completed", "partially_refunded", "cancelled", "fully_refunded":
		return true
	default:
		return false
	}
}

func (s *AdminDonationService) GetOrder(seq int64) (*model.DonationOrder, error) {
	if seq <= 0 {
		return nil, fmt.Errorf("%w: invalid donation order sequence", ErrInvalidDonationOrder)
	}
	return s.repo.GetDonationOrder(seq)
}

func (s *AdminDonationService) CreateOrder(input model.DonationOrderInput, operSeq int, ip string) (*model.DonationOrder, error) {
	normalized, err := NormalizeDonationOrderInput(input)
	if err != nil {
		return nil, err
	}
	if err := validateDonationAccountInput(normalized.AccountUsrSeq); err != nil {
		return nil, err
	}
	seq, err := s.repo.CreateDonationOrder(normalized, operSeq, ip)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDonationOrder(seq)
}

func (s *AdminDonationService) UpdateOrder(seq int64, input model.DonationOrderInput, operSeq int, ip string) (*model.DonationOrder, error) {
	if seq <= 0 {
		return nil, fmt.Errorf("%w: invalid donation order sequence", ErrInvalidDonationOrder)
	}
	normalized, err := NormalizeDonationOrderInput(input)
	if err != nil {
		return nil, err
	}
	if normalized.AccountUsrSeqSet {
		if err := validateDonationAccountInput(normalized.AccountUsrSeq); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateDonationOrder(seq, normalized, operSeq, ip); err != nil {
		return nil, err
	}
	return s.repo.GetDonationOrder(seq)
}

func validateDonationAccountInput(accountUsrSeq *int) error {
	if accountUsrSeq == nil {
		return nil
	}
	if *accountUsrSeq <= 0 {
		return fmt.Errorf("%w: invalid account user sequence", ErrInvalidDonationOrder)
	}
	return nil
}

func importedDonationOrderInput(row model.ImportedDonationRow, accountUsrSeq *int) model.DonationOrderInput {
	return model.DonationOrderInput{
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
	}
}

func normalizeImportedDonationOrder(row model.ImportedDonationRow, accountUsrSeq *int) (model.NormalizedDonationOrder, error) {
	return NormalizeDonationOrderInput(importedDonationOrderInput(row, accountUsrSeq))
}

// CreateImportedOrderTx intentionally follows the same normalization and
// repository writer as the ordinary administrator create path.
func (s *AdminDonationService) CreateImportedOrderTx(tx *sqlx.Tx, row model.ImportedDonationRow, accountUsrSeq *int, operSeq int, ip string) (int64, error) {
	normalized, err := normalizeImportedDonationOrder(row, accountUsrSeq)
	if err != nil {
		return 0, err
	}
	if err := validateDonationAccountInput(normalized.AccountUsrSeq); err != nil {
		return 0, err
	}
	return s.repo.CreateDonationOrderTx(tx, normalized, operSeq, ip)
}

func (s *AdminDonationService) CreateImportedOrdersTx(tx *sqlx.Tx, orders []model.ImportedDonationOrder, operSeq int, ip string) ([]int64, error) {
	normalizedOrders := make([]model.NormalizedDonationOrder, 0, len(orders))
	for _, order := range orders {
		normalized, err := normalizeImportedDonationOrder(order.ImportedDonationRow, order.AccountUsrSeq)
		if err != nil {
			return nil, err
		}
		if err := validateDonationAccountInput(normalized.AccountUsrSeq); err != nil {
			return nil, err
		}
		normalizedOrders = append(normalizedOrders, normalized)
	}
	return s.repo.CreateDonationOrdersTx(tx, normalizedOrders, operSeq, ip)
}
