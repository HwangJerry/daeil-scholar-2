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
	if snapshot == nil {
		summary, err := s.computeLiveSummary()
		if err != nil {
			return nil, err
		}
		s.cache.Set("donation_summary", summary, 5*time.Minute)
		return summary, nil
	}

	config, err := s.repo.GetActiveConfig()
	if err != nil {
		return nil, err
	}
	displayAmount := snapshot.DSTotal + snapshot.ManualAdj
	donorCount := snapshot.DonorCnt
	if snapshot.Overwrite == "Y" {
		displayAmount = snapshot.ManualAdj
		if config != nil {
			donorCount = config.ManualDonorCnt
		}
	}
	achievementRate := float64(0)
	if snapshot.Goal > 0 {
		achievementRate = float64(displayAmount) / float64(snapshot.Goal) * 100
	}
	summary := &model.DonationSummary{
		DisplayAmount:   displayAmount,
		GoalAmount:      snapshot.Goal,
		DonorCount:      donorCount,
		AchievementRate: achievementRate,
		SnapshotDate:    snapshot.DSDate,
	}
	if config != nil {
		summary.TierThresholds = model.DonationTierThresholds{
			Sprout:   config.TierSproutMin,
			Sapling:  config.TierSaplingMin,
			Tree:     config.TierTreeMin,
			Blooming: config.TierBloomingMin,
			Fruiting: config.TierFruitingMin,
		}
	}
	s.cache.Set("donation_summary", summary, 5*time.Minute)
	return summary, nil
}

// InvalidateCache evicts the cached donation summary so the next call recomputes from the snapshot.
func (s *DonationService) InvalidateCache() {
	s.cache.Delete("donation_summary")
}

func (s *DonationService) computeLiveSummary() (*model.DonationSummary, error) {
	total, donorCount, err := s.repo.GetReceivedDonationAggregate()
	if err != nil {
		return nil, err
	}
	config, err := s.repo.GetActiveConfig()
	if err != nil {
		return nil, err
	}

	manualAdj := int64(0)
	goal := int64(0)
	tierThresholds := model.DonationTierThresholds{}
	if config != nil {
		manualAdj = config.ManualAdj
		goal = config.Goal
		tierThresholds = model.DonationTierThresholds{
			Sprout:   config.TierSproutMin,
			Sapling:  config.TierSaplingMin,
			Tree:     config.TierTreeMin,
			Blooming: config.TierBloomingMin,
			Fruiting: config.TierFruitingMin,
		}
	}

	displayAmount := total + manualAdj
	if config != nil && config.Overwrite == "Y" {
		displayAmount = manualAdj
		donorCount = config.ManualDonorCnt
	}
	achievementRate := float64(0)
	if goal > 0 {
		achievementRate = float64(displayAmount) / float64(goal) * 100
	}

	return &model.DonationSummary{
		DisplayAmount:   displayAmount,
		GoalAmount:      goal,
		DonorCount:      donorCount,
		AchievementRate: achievementRate,
		SnapshotDate:    time.Now().Format("2006-01-02"),
		TierThresholds:  tierThresholds,
	}, nil
}
