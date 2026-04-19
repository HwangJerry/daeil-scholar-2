// Admin donation service — business logic for donation config
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type AdminDonationService struct {
	repo         *repository.AdminDonationRepository
	donationRepo *repository.DonationRepository
}

func NewAdminDonationService(repo *repository.AdminDonationRepository, donationRepo *repository.DonationRepository) *AdminDonationService {
	return &AdminDonationService{repo: repo, donationRepo: donationRepo}
}

func (s *AdminDonationService) GetConfig() (*model.DonationConfig, error) {
	return s.donationRepo.GetActiveConfig()
}

func (s *AdminDonationService) UpdateConfig(goal int64, manualAdj int64, note string, overwrite bool, operSeq int) error {
	ov := "N"
	if overwrite {
		ov = "Y"
	}
	return s.repo.UpdateConfig(goal, manualAdj, note, ov, operSeq)
}

func (s *AdminDonationService) GetHistory(days int) ([]model.DonationSnapshot, error) {
	if days <= 0 {
		days = 30
	}
	return s.repo.GetSnapshotHistory(days)
}

func (s *AdminDonationService) ListOrders(page, size int, search, status, payType string) ([]model.AdminDonationOrderRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	if status != "" && status != "Y" && status != "N" {
		return nil, 0, fmt.Errorf("invalid payment status: %s", status)
	}
	return s.repo.GetDonationOrders(page, size, search, status, payType)
}

func (s *AdminDonationService) UpdateOrder(seq int, payment string, amount int) error {
	if payment != "Y" && payment != "N" {
		return fmt.Errorf("invalid payment value: %s", payment)
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	return s.repo.UpdateDonationOrder(seq, payment, amount)
}
