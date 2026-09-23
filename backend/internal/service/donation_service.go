package service

import (
	"sync/atomic"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

type DonationService struct {
	repo          *repository.DonationRepository
	cache         *cache.Cache
	snapshotStale atomic.Bool
}

type cachedDonationSummary struct {
	month   string
	summary *model.DonationSummary
}

func NewDonationService(repo *repository.DonationRepository, cacheStore *cache.Cache) *DonationService {
	return &DonationService{repo: repo, cache: cacheStore}
}

func (s *DonationService) GetSummary() (*model.DonationSummary, error) {
	return s.getSummaryAt(time.Now())
}

func (s *DonationService) getSummaryAt(now time.Time) (*model.DonationSummary, error) {
	seoul, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return nil, err
	}
	now = now.In(seoul)
	month := now.Format("2006-01")
	if !s.snapshotStale.Load() {
		if cached, found := s.cache.Get("donation_summary"); found {
			if entry, ok := cached.(cachedDonationSummary); ok && entry.month == month {
				return entry.summary, nil
			}
		}
	}

	summary, err := s.computeSummary(now)
	if err != nil {
		return nil, err
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, seoul)
	summary.MonthAmount, err = s.repo.GetReceivedDonationAmountBetween(start, start.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	// Validate the calendar month on cache reads even within the five-minute TTL.
	s.cache.Set("donation_summary", cachedDonationSummary{month: month, summary: summary}, 5*time.Minute)
	return summary, nil
}

func (s *DonationService) computeSummary(now time.Time) (*model.DonationSummary, error) {
	if s.snapshotStale.Load() {
		return s.computeLiveSummary(now)
	}
	snapshot, err := s.repo.GetSnapshotByDate(now)
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
		return s.computeLiveSummary(now)
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
		summary.BalanceAmount = config.BalanceAmount
		summary.BalanceAsOf = config.BalanceAsOf
		summary.TierThresholds = model.DonationTierThresholds{
			Sprout:   config.TierSproutMin,
			Sapling:  config.TierSaplingMin,
			Tree:     config.TierTreeMin,
			Blooming: config.TierBloomingMin,
			Fruiting: config.TierFruitingMin,
		}
	}
	return summary, nil
}

// InvalidateCache evicts the cached donation summary so the next call recomputes from the snapshot.
func (s *DonationService) InvalidateCache() {
	s.cache.Delete("donation_summary")
}

func (s *DonationService) MarkSnapshotFresh() {
	s.snapshotStale.Store(false)
	s.InvalidateCache()
}

func (s *DonationService) MarkSnapshotStale() {
	s.snapshotStale.Store(true)
	s.InvalidateCache()
}

func (s *DonationService) computeLiveSummary(now time.Time) (*model.DonationSummary, error) {
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

	summary := &model.DonationSummary{
		DisplayAmount:   displayAmount,
		GoalAmount:      goal,
		DonorCount:      donorCount,
		AchievementRate: achievementRate,
		SnapshotDate:    now.Format("2006-01-02"),
		TierThresholds:  tierThresholds,
	}
	if config != nil {
		summary.BalanceAmount = config.BalanceAmount
		summary.BalanceAsOf = config.BalanceAsOf
	}
	return summary, nil
}
