// visit_aggregation.go — Daily job aggregating WEO_VISIT_DAILY into WEO_VISIT_SUMMARY and pruning old rows
package job

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

// kstZone pins date rollovers to Asia/Seoul so the 00:10 trigger is stable
// regardless of the host machine's TZ.
var kstZone = time.FixedZone("KST", 9*3600)

const (
	visitRetentionDays = 90
	backfillWindowDays = 7
)

// VisitAggregationJob produces a per-day summary row and prunes raw visit
// records older than the retention window. Modeled after DonationSnapshotJob.
type VisitAggregationJob struct {
	repo   *repository.VisitRepository
	logger zerolog.Logger
	cancel context.CancelFunc
}

func NewVisitAggregationJob(repo *repository.VisitRepository, logger zerolog.Logger) *VisitAggregationJob {
	return &VisitAggregationJob{repo: repo, logger: logger}
}

// Start backfills any missing summary rows for the last few days, then loops
// forever writing the prior day's summary at 00:10 KST.
func (j *VisitAggregationJob) Start() {
	j.backfillRecent()

	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go func() {
		defer func() {
			if r := recover(); r != nil {
				j.logger.Error().Interface("panic", r).Msg("visit aggregation job panicked")
			}
		}()
		for {
			next := nextRunAt(time.Now().In(kstZone))
			select {
			case <-ctx.Done():
				j.logger.Info().Msg("visit aggregation job stopped")
				return
			case <-time.After(time.Until(next)):
				yesterday := next.AddDate(0, 0, -1)
				if err := j.aggregate(yesterday); err != nil {
					j.logger.Error().Err(err).Msg("visit aggregation failed")
				} else {
					j.logger.Info().Str("date", yesterday.Format("2006-01-02")).Msg("visit summary written")
				}
				if err := j.prune(next); err != nil {
					j.logger.Error().Err(err).Msg("visit prune failed")
				}
			}
		}
	}()
}

func (j *VisitAggregationJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}

// backfillRecent ensures the last N days have summary rows so the admin chart
// isn't empty after a server restart that missed a midnight window.
func (j *VisitAggregationJob) backfillRecent() {
	now := time.Now().In(kstZone)
	for offset := 1; offset <= backfillWindowDays; offset++ {
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, kstZone).AddDate(0, 0, -offset)
		exists, err := j.repo.HasSummary(day)
		if err != nil {
			j.logger.Error().Err(err).Msg("backfill check failed")
			continue
		}
		if exists {
			continue
		}
		if err := j.aggregate(day); err != nil {
			j.logger.Error().Err(err).Str("date", day.Format("2006-01-02")).Msg("backfill aggregate failed")
			continue
		}
		j.logger.Info().Str("date", day.Format("2006-01-02")).Msg("visit summary backfilled")
	}
}

// aggregate computes and upserts one day's summary row.
func (j *VisitAggregationJob) aggregate(day time.Time) error {
	total, member, anon, err := j.repo.CountDAU(day)
	if err != nil {
		return err
	}
	mau, err := j.repo.CountMAU(day)
	if err != nil {
		return err
	}
	pv, err := j.repo.SumPageViews(day)
	if err != nil {
		return err
	}
	return j.repo.UpsertSummary(model.VisitSummary{
		Date:      day,
		DAUTotal:  total,
		DAUMember: member,
		DAUAnon:   anon,
		MAUTotal:  mau,
		PageViews: pv,
		RegDate:   time.Now().In(kstZone),
	})
}

// prune deletes raw visit rows older than the retention window.
func (j *VisitAggregationJob) prune(now time.Time) error {
	cutoff := now.AddDate(0, 0, -visitRetentionDays)
	deleted, err := j.repo.DeleteDailyBefore(cutoff)
	if err != nil {
		return err
	}
	if deleted > 0 {
		j.logger.Info().Int64("count", deleted).Msg("pruned old visit rows")
	}
	return nil
}

// nextRunAt returns the next 00:10 KST moment strictly after now.
func nextRunAt(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 10, 0, 0, kstZone)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
