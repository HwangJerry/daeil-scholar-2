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
	snapshot, err := s.repo.GetSnapshotByDate(time.Now())
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		snapshot, err = s.repo.GetLatestSnapshot()
		if err != nil {
			return nil, err
		}
	}
	// Fallback: if no snapshot exists, compute live from WEO_ORDER + DONATION_CONFIG
	if snapshot == nil {
		summary, err := s.computeLiveSummary()
		if err != nil {
			return nil, err
		}
		s.cache.Set("donation_summary", summary, 5*time.Minute)
		return summary, nil
	}

	displayAmount := snapshot.DSTotal + snapshot.ManualAdj
	if snapshot.Overwrite == "Y" {
		displayAmount = snapshot.ManualAdj
	}
	rate := float64(0)
	if snapshot.Goal > 0 {
		rate = float64(displayAmount) / float64(snapshot.Goal) * 100
	}
	summary := &model.DonationSummary{
		TotalAmount:     snapshot.DSTotal,
		ManualAdj:       snapshot.ManualAdj,
		DisplayAmount:   displayAmount,
		DonorCount:      snapshot.DonorCnt,
		GoalAmount:      snapshot.Goal,
		AchievementRate: rate,
		SnapshotDate:    snapshot.DSDate,
	}
	s.cache.Set("donation_summary", summary, 5*time.Minute)
	return summary, nil
}

// InvalidateCache evicts the cached donation summary so the next call recomputes from the snapshot.
func (s *DonationService) InvalidateCache() {
	s.cache.Delete("donation_summary")
}

func (s *DonationService) computeLiveSummary() (*model.DonationSummary, error) {
	total, err := s.repo.SumDonations()
	if err != nil {
		return nil, err
	}
	donorCount, err := s.repo.CountDonors()
	if err != nil {
		return nil, err
	}
	config, err := s.repo.GetActiveConfig()
	if err != nil {
		return nil, err
	}

	manualAdj := int64(0)
	goal := int64(0)
	if config != nil {
		manualAdj = config.ManualAdj
		goal = config.Goal
	}

	displayAmount := total + manualAdj
	if config != nil && config.Overwrite == "Y" {
		displayAmount = manualAdj
	}
	rate := float64(0)
	if goal > 0 {
		rate = float64(displayAmount) / float64(goal) * 100
	}

	return &model.DonationSummary{
		TotalAmount:     total,
		ManualAdj:       manualAdj,
		DisplayAmount:   displayAmount,
		DonorCount:      donorCount,
		GoalAmount:      goal,
		AchievementRate: rate,
		SnapshotDate:    time.Now().Format("2006-01-02"),
	}, nil
}
