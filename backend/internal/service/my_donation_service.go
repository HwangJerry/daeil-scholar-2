// my_donation_service.go — business logic for user donation history
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type MyDonationService struct {
	repo *repository.MyDonationRepository
}

func NewMyDonationService(repo *repository.MyDonationRepository) *MyDonationService {
	return &MyDonationService{repo: repo}
}

func (s *MyDonationService) GetMyDonations(usrSeq int, sort string, page int, size int) (*model.MyDonationResponse, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	if sort != "amount" {
		sort = "latest"
	}

	items, totalCount, err := s.repo.GetMyDonations(usrSeq, sort, page, size)
	if err != nil {
		return nil, err
	}

	totalAmount, err := s.repo.GetMyTotalDonation(usrSeq)
	if err != nil {
		return nil, err
	}

	return &model.MyDonationResponse{
		Items:       items,
		TotalAmount: totalAmount,
		TotalCount:  totalCount,
	}, nil
}
