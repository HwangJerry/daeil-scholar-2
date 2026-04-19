// Admin ad service — business logic for ad management
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// ErrTierConflict is returned when PREMIUM or GOLD already has an active ad.
var ErrTierConflict = errors.New("tier_conflict")

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
	if (a.ADTier == "PREMIUM" || a.ADTier == "GOLD") && a.MAStatus == "Y" {
		count, err := s.repo.CountActiveTierAds(a.ADTier, 0)
		if err != nil {
			return 0, err
		}
		if count > 0 {
			return 0, ErrTierConflict
		}
	}
	return s.repo.InsertAd(a)
}

func (s *AdminAdService) Update(seq int, a *model.AdminAdInsert) error {
	if (a.ADTier == "PREMIUM" || a.ADTier == "GOLD") && a.MAStatus == "Y" {
		count, err := s.repo.CountActiveTierAds(a.ADTier, seq)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrTierConflict
		}
	}
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
