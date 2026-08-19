// DonationConfigOrchestrator — single entry point for all admin donation operations
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// snapshotCreator is satisfied by job.DonationSnapshotJob.
type snapshotCreator interface {
	CreateSnapshotNow() error
	CreateSnapshotTx(tx *sqlx.Tx) error
}

// DonationConfigOrchestrator is the single service the admin donation handler calls.
// Simple read/write operations are delegated to AdminDonationService.
// UpdateConfig additionally refreshes today's snapshot and evicts the cache so
// the public summary reflects the new values immediately.
type DonationConfigOrchestrator struct {
	adminSvc    *AdminDonationService
	donationSvc *DonationService
	snapshotJob snapshotCreator
}

func NewDonationConfigOrchestrator(
	adminSvc *AdminDonationService,
	donationSvc *DonationService,
	snapshotJob snapshotCreator,
) *DonationConfigOrchestrator {
	return &DonationConfigOrchestrator{
		adminSvc:    adminSvc,
		donationSvc: donationSvc,
		snapshotJob: snapshotJob,
	}
}

func (o *DonationConfigOrchestrator) GetConfig() (*model.DonationConfig, error) {
	return o.adminSvc.GetConfig()
}

// UpdateConfig persists the config, refreshes today's snapshot, and invalidates the cache.
func (o *DonationConfigOrchestrator) UpdateConfig(update DonationConfigUpdate, operSeq int) error {
	if !validDonationTierThresholds(update) {
		return ErrInvalidTierThresholds
	}
	err := o.adminSvc.RunInTransaction(func(tx *sqlx.Tx) error {
		if err := o.adminSvc.UpdateConfigTx(tx, update, operSeq); err != nil {
			return err
		}
		if err := o.snapshotJob.CreateSnapshotTx(tx); err != nil {
			return fmt.Errorf("snapshot refresh: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("config and snapshot update failed: %w", err)
	}
	o.donationSvc.MarkSnapshotFresh()
	return nil
}

// RefreshDonationSummary recomputes today's snapshot and updates the public
// summary cache policy. If snapshot persistence fails, the next public read
// uses the canonical live aggregate instead of serving stale snapshot data.
func (o *DonationConfigOrchestrator) RefreshDonationSummary() error {
	if err := o.snapshotJob.CreateSnapshotNow(); err != nil {
		o.donationSvc.MarkSnapshotStale()
		return err
	}
	o.donationSvc.MarkSnapshotFresh()
	return nil
}

func (o *DonationConfigOrchestrator) GetHistory(days int) ([]model.DonationSnapshot, error) {
	return o.adminSvc.GetHistory(days)
}

func (o *DonationConfigOrchestrator) ListOrders(filters model.DonationOrderFilters, page, size int) (*model.DonationOrderPage, error) {
	return o.adminSvc.ListOrders(filters, page, size)
}

func (o *DonationConfigOrchestrator) GetOrder(seq int64) (*model.DonationOrder, error) {
	return o.adminSvc.GetOrder(seq)
}

func (o *DonationConfigOrchestrator) CreateOrder(input model.DonationOrderInput, operSeq int, ip string) (*model.DonationOrder, error) {
	order, err := o.adminSvc.CreateOrder(input, operSeq, ip)
	if err != nil {
		return nil, err
	}
	_ = o.RefreshDonationSummary()
	return order, nil
}

func (o *DonationConfigOrchestrator) UpdateOrder(seq int64, input model.DonationOrderInput, operSeq int, ip string) (*model.DonationOrder, error) {
	order, err := o.adminSvc.UpdateOrder(seq, input, operSeq, ip)
	if err != nil {
		return nil, err
	}
	_ = o.RefreshDonationSummary()
	return order, nil
}
