// subscription_billing.go — Daily KST 03:00 batch: charge active subscriptions via stored billing keys
package job

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// MaxConsecutiveFails is the threshold at which a subscription is auto-suspended.
const MaxConsecutiveFails = 3

// AutoBillingClock is the time of day (KST) the daily run fires.
var (
	AutoBillingHour   = 3
	AutoBillingMinute = 0
)

// EasyPayBilling is the subset of EasyPay behaviour the job depends on. Lets unit tests
// stub PG calls without spawning ep_cli.
type EasyPayBilling interface {
	AutoBilling(billingKey, orderNo string, amount int, traceNo, clientIP string) (*model.ApproveResult, error)
}

// SubscriptionBillingJob charges active recurring subscriptions whose BILL_DAY matches today.
// Each subscription is processed in its own transaction; one failure does not roll back others.
type SubscriptionBillingJob struct {
	subRepo    *repository.SubscriptionRepository
	donateRepo *repository.DonateRepository
	epSvc      EasyPayBilling
	pgAudit    *service.PGAuditLogger
	cache      *cache.Cache
	cfg        config.EasyPayConfig
	logger     zerolog.Logger
	cancel     context.CancelFunc
}

// NewSubscriptionBillingJob wires the daily billing job.
func NewSubscriptionBillingJob(
	subRepo *repository.SubscriptionRepository,
	donateRepo *repository.DonateRepository,
	epSvc EasyPayBilling,
	pgAudit *service.PGAuditLogger,
	cacheStore *cache.Cache,
	cfg config.EasyPayConfig,
	logger zerolog.Logger,
) *SubscriptionBillingJob {
	return &SubscriptionBillingJob{
		subRepo:    subRepo,
		donateRepo: donateRepo,
		epSvc:      epSvc,
		pgAudit:    pgAudit,
		cache:      cacheStore,
		cfg:        cfg,
		logger:     logger,
	}
}

// Start launches the daily ticker. Mirrors job.DonationSnapshotJob.Start: cancellable
// context + goroutine + select on time.After(timeUntilNext03KST).
func (j *SubscriptionBillingJob) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go func() {
		defer func() {
			if r := recover(); r != nil {
				j.logger.Error().Interface("panic", r).Msg("subscription billing job panicked")
			}
		}()
		for {
			next := nextRunAt(time.Now())
			select {
			case <-ctx.Done():
				j.logger.Info().Msg("subscription billing job stopped")
				return
			case <-time.After(time.Until(next)):
				processed, errs := j.RunOnce(time.Now())
				if len(errs) > 0 {
					j.logger.Warn().Int("processed", processed).Int("errors", len(errs)).Msg("subscription billing run finished with errors")
				} else {
					j.logger.Info().Int("processed", processed).Msg("subscription billing run finished")
				}
			}
		}
	}()
}

// Stop signals the running goroutine to exit on its next tick.
func (j *SubscriptionBillingJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}

// RunOnce charges every active subscription whose BILL_DAY matches `now.Day()` and that has
// not already been billed in the current month. Safe to call from the daily loop or from the
// admin manual-trigger endpoint. Returns the count of attempted-and-succeeded charges plus
// the slice of per-subscription errors (one entry per failing row).
func (j *SubscriptionBillingJob) RunOnce(now time.Time) (int, []error) {
	billDay := now.Day()
	if billDay > 28 {
		billDay = 28
	}
	currentYYYYMM := now.Format("200601")

	due, err := j.subRepo.ListDueForBilling(billDay, currentYYYYMM)
	if err != nil {
		j.logger.Error().Err(err).Msg("subscription billing: list due failed")
		return 0, []error{err}
	}

	processed := 0
	var errs []error
	for _, sub := range due {
		if err := j.safeProcessOne(sub, now); err != nil {
			errs = append(errs, fmt.Errorf("sub %d: %w", sub.SubSeq, err))
			continue
		}
		processed++
	}
	if processed > 0 {
		j.cache.Delete("donation_summary")
	}
	return processed, errs
}

// safeProcessOne wraps processOne with a recover so a panic on one subscription cannot
// abort the whole batch.
func (j *SubscriptionBillingJob) safeProcessOne(sub *model.Subscription, now time.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
			j.logger.Error().Interface("panic", r).Int("subSeq", sub.SubSeq).Msg("subscription billing: panic in processOne")
		}
	}()
	return j.processOne(sub, now)
}

// processOne charges a single subscription. Steps (all PG-related events go through pgAudit):
//
//  1. Sanity check: billing key must be present.
//  2. Insert WEO_ORDER row (gate='P', payType='CARD').
//  3. Call EasyPay AutoBilling with the stored billing key.
//  4. On success: insert WEO_PG_DATA, update WEO_ORDER, mark subscription billed,
//     all in a single transaction.
//  5. On failure: bump fail count; suspend if it reaches MaxConsecutiveFails.
func (j *SubscriptionBillingJob) processOne(sub *model.Subscription, now time.Time) error {
	if !sub.BillingKey.Valid || sub.BillingKey.String == "" {
		j.logger.Warn().Int("subSeq", sub.SubSeq).Msg("subscription billing: missing billing key — skipping")
		return errors.New("missing billing key")
	}

	orderSeq, err := j.donateRepo.InsertOrder(sub.USRSeq, "P", sub.PayType, sub.Amount, "auto-billing")
	if err != nil {
		j.logger.Error().Err(err).Int("subSeq", sub.SubSeq).Msg("subscription billing: order insert failed")
		return err
	}
	orderNo := strconv.FormatInt(orderSeq, 10)
	traceNo := fmt.Sprintf("auto-%d-%d", sub.SubSeq, now.UnixNano()/int64(time.Millisecond))

	result, err := j.epSvc.AutoBilling(sub.BillingKey.String, orderNo, sub.Amount, traceNo, "auto-billing")
	if err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_fail", nil, err)
		j.handleFailure(sub, err)
		return err
	}
	if result.ResCode != "0000" {
		approveErr := fmt.Errorf("PG 승인 실패: %s (%s)", result.ResMsg, result.ResCode)
		j.pgAudit.Log(orderNo, "auto_billing_fail", result, nil)
		j.handleFailure(sub, approveErr)
		return approveErr
	}
	j.pgAudit.Log(orderNo, "auto_billing_success", result, nil)

	approvedAmount, _ := strconv.Atoi(result.Amount)
	if approvedAmount != sub.Amount {
		mismatchErr := fmt.Errorf("amount mismatch: subscription %d, approved %d", sub.Amount, approvedAmount)
		j.pgAudit.Log(orderNo, "auto_billing_amount_mismatch", result, mismatchErr)
		j.handleFailure(sub, mismatchErr)
		return mismatchErr
	}

	if err := j.commitChargeTx(sub, int(orderSeq), approvedAmount, result, orderNo, now); err != nil {
		j.handleFailure(sub, err)
		return err
	}
	return nil
}

// commitChargeTx persists PG_DATA + ORDER update + SUBSCRIPTION billed-marker in one tx.
func (j *SubscriptionBillingJob) commitChargeTx(
	sub *model.Subscription,
	orderSeq, amount int,
	result *model.ApproveResult,
	orderNo string,
	now time.Time,
) error {
	tx, err := j.subRepo.DB.Beginx()
	if err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_db_tx_begin_fail", result, err)
		return fmt.Errorf("tx begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := j.persistPGAndOrderTx(tx, orderSeq, amount, result, orderNo); err != nil {
		return err
	}

	billDay := int(sub.BillDay.Int64)
	nextBill := service.ComputeNextBillDate(now, billDay)
	if err := j.subRepo.MarkBilledTx(tx, sub.SubSeq, now, nextBill); err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_db_sub_update_fail", result, err)
		return fmt.Errorf("sub update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_db_tx_commit_fail", result, err)
		return fmt.Errorf("tx commit: %w", err)
	}
	j.logger.Info().Int("subSeq", sub.SubSeq).Int("amount", amount).Str("orderNo", orderNo).Msg("auto-billing charged")
	return nil
}

// persistPGAndOrderTx writes the PG approval row and flips the order to paid.
func (j *SubscriptionBillingJob) persistPGAndOrderTx(
	tx *sqlx.Tx,
	orderSeq, amount int,
	result *model.ApproveResult,
	orderNo string,
) error {
	pgData := &model.PGData{
		CNO:      result.CNO,
		ResCD:    result.ResCode,
		ResMsg:   result.ResMsg,
		Amount:   amount,
		NumCard:  result.CardNo,
		TranDate: result.TranDate,
		AuthNo:   result.AuthNo,
		PayType:  result.PayType,
		OSeq:     orderSeq,
	}
	pgSeq, err := j.donateRepo.InsertPGDataTx(tx, pgData)
	if err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_db_pg_insert_fail", result, err)
		return fmt.Errorf("pg insert: %w", err)
	}
	if _, err := j.donateRepo.UpdateOrderPaymentTx(tx, orderSeq, amount, pgSeq, "auto-billing"); err != nil {
		j.pgAudit.Log(orderNo, "auto_billing_db_order_update_fail", result, err)
		return fmt.Errorf("order update: %w", err)
	}
	return nil
}

// handleFailure increments fail count and, on reaching the consecutive-fail threshold,
// suspends the subscription with a WARN log + audit entry.
func (j *SubscriptionBillingJob) handleFailure(sub *model.Subscription, cause error) {
	failCount, incErr := j.subRepo.IncrementFailCount(sub.SubSeq)
	if incErr != nil {
		j.logger.Error().Err(incErr).Int("subSeq", sub.SubSeq).Msg("subscription billing: fail count increment failed")
		return
	}
	if failCount >= MaxConsecutiveFails {
		if err := j.subRepo.MarkSuspended(sub.SubSeq); err != nil {
			j.logger.Error().Err(err).Int("subSeq", sub.SubSeq).Msg("subscription billing: suspend failed")
			return
		}
		j.logger.Warn().Int("subSeq", sub.SubSeq).Int("failCount", failCount).Err(cause).Msg("subscription auto-suspended after consecutive failures")
		j.pgAudit.Log(strconv.Itoa(sub.SubSeq), "auto_billing_suspended", map[string]int{"failCount": failCount}, cause)
	}
}

// nextRunAt returns the next KST 03:00 strictly after `now`. If `now` is before today's
// 03:00 in its location, today's 03:00 is returned; otherwise tomorrow's.
func nextRunAt(now time.Time) time.Time {
	loc := now.Location()
	candidate := time.Date(now.Year(), now.Month(), now.Day(), AutoBillingHour, AutoBillingMinute, 0, 0, loc)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}
