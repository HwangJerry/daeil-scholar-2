package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type BannerAdService struct {
	repo *repository.BannerAdRepository
}

func NewBannerAdService(repo *repository.BannerAdRepository) *BannerAdService {
	return &BannerAdService{repo: repo}
}

func (s *BannerAdService) GetActiveBanner() (*model.BannerAd, error) {
	return s.repo.GetActiveBanner()
}

func (s *BannerAdService) LogEvent(bnSeq int, eventType string) error {
	return s.repo.LogEvent(bnSeq, eventType)
}
