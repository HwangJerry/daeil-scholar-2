package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

const (
	defaultPersonalDonationPageSize = 20
	maxPersonalDonationPageSize     = 50
)

var ErrInvalidPersonalDonationUser = errors.New("invalid personal donation user")

type personalDonationRepository interface {
	GetTotals(usrSeq int) (int64, int, error)
	List(usrSeq int, sort string, page, size int) ([]model.PersonalDonationItem, error)
}

type PersonalDonationService struct {
	repo personalDonationRepository
}

func NewPersonalDonationService(repo personalDonationRepository) *PersonalDonationService {
	return &PersonalDonationService{repo: repo}
}

func (s *PersonalDonationService) GetPersonalDonations(usrSeq int, sort string, page, size int) (*model.PersonalDonationSummary, error) {
	if usrSeq <= 0 {
		return nil, ErrInvalidPersonalDonationUser
	}
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > maxPersonalDonationPageSize {
		size = defaultPersonalDonationPageSize
	}
	if sort != "amount" {
		sort = "latest"
	}

	totalAmount, totalCount, err := s.repo.GetTotals(usrSeq)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.List(usrSeq, sort, page, size)
	if err != nil {
		return nil, err
	}

	return &model.PersonalDonationSummary{
		Items:       items,
		TotalAmount: totalAmount,
		TotalCount:  totalCount,
		Page:        page,
		Size:        size,
	}, nil
}
