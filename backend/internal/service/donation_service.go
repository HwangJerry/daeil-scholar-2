package service

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

type DonationService struct {
	repo  *repository.DonationRepository
	cache *cache.Cache
}

func NewDonationService(repo *repository.DonationRepository, cacheStore *cache.Cache) *DonationService {
	return &DonationService{repo: repo, cache: cacheStore}
}

func (s *DonationService) GetSummary() (*model.DonationSummary, error) {
	if cached, found := s.cache.Get("donation_summary"); found {
		if summary, ok := cached.(*model.DonationSummary); ok {
			return summary, nil
		}
	}
	total, err := s.repo.SumNetReceivedDonations()
	if err != nil {
		return nil, err
	}
	summary := &model.DonationSummary{DisplayAmount: total}
	s.cache.Set("donation_summary", summary, 5*time.Minute)
	return summary, nil
}

// InvalidateCache evicts the cached donation summary so the next call recomputes from the ledger.
func (s *DonationService) InvalidateCache() {
	s.cache.Delete("donation_summary")
}
