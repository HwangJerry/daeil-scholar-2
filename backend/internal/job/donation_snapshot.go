package job

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

type DonationSnapshotJob struct {
	repo   *repository.DonationRepository
	logger zerolog.Logger
	cancel context.CancelFunc
}

func NewDonationSnapshotJob(repo *repository.DonationRepository, logger zerolog.Logger) *DonationSnapshotJob {
	return &DonationSnapshotJob{repo: repo, logger: logger}
}

func (j *DonationSnapshotJob) Start() {
	if exists, err := j.repo.HasSnapshotForDate(time.Now()); err == nil && !exists {
		if err := j.createSnapshot(); err != nil {
			j.logger.Error().Err(err).Msg("startup snapshot backfill failed")
		} else {
			j.logger.Info().Msg("startup snapshot backfill completed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go func() {
		defer func() {
			if r := recover(); r != nil {
				j.logger.Error().Interface("panic", r).Msg("donation snapshot job panicked")
			}
		}()
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
			select {
			case <-ctx.Done():
				j.logger.Info().Msg("donation snapshot job stopped")
				return
			case <-time.After(time.Until(next)):
				if err := j.createSnapshot(); err != nil {
					j.logger.Error().Err(err).Msg("donation snapshot failed")
				} else {
					j.logger.Info().Msg("donation snapshot created")
				}
			}
		}
	}()
}

func (j *DonationSnapshotJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}

func (j *DonationSnapshotJob) createSnapshot() error {
	total, err := j.repo.SumDonations()
	if err != nil {
		return err
	}
	donorCount, err := j.repo.CountDonors()
	if err != nil {
		return err
	}
	config, err := j.repo.GetActiveConfig()
	if err != nil {
		return err
	}
	manualAdj := int64(0)
	goal := int64(0)
	if config != nil {
		manualAdj = config.ManualAdj
		goal = config.Goal
	}
	return j.repo.UpsertSnapshot(time.Now(), total, manualAdj, donorCount, goal)
}
