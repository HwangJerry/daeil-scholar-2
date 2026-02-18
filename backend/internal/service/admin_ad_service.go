// Admin ad service — business logic for ad management
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type AdminAdService struct {
	repo *repository.AdminAdRepository
}

func NewAdminAdService(repo *repository.AdminAdRepository) *AdminAdService {
	return &AdminAdService{repo: repo}
}

func (s *AdminAdService) List() ([]model.AdminAdRow, error) {
	return s.repo.GetAds()
}

func (s *AdminAdService) Create(a *model.AdminAdInsert) (int, error) {
	return s.repo.InsertAd(a)
}

func (s *AdminAdService) Update(seq int, a *model.AdminAdInsert) error {
	return s.repo.UpdateAd(seq, a)
}

func (s *AdminAdService) Delete(seq int) error {
	return s.repo.DeleteAd(seq)
}

func (s *AdminAdService) GetStats(maSeq int) (*model.AdminAdStats, error) {
	return s.repo.GetAdStats(maSeq)
}

func (s *AdminAdService) GetDashboardAdStats() (*model.DashboardAdStats, error) {
	impressions, err := s.repo.GetTotalImpressions()
	if err != nil {
		return nil, err
	}
	clicks, err := s.repo.GetTotalClicks()
	if err != nil {
		return nil, err
	}
	ctr := float64(0)
	if impressions > 0 {
		ctr = float64(clicks) / float64(impressions) * 100
	}
	return &model.DashboardAdStats{
		TotalImpressions: impressions,
		TotalClicks:      clicks,
		CTR:              ctr,
	}, nil
}
