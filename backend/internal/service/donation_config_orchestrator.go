// DonationConfigOrchestrator — single entry point for all admin donation operations
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
)

// snapshotCreator is satisfied by job.DonationSnapshotJob.
type snapshotCreator interface {
	CreateSnapshotNow() error
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
func (o *DonationConfigOrchestrator) UpdateConfig(goal int64, manualAdj int64, note string, overwrite bool, operSeq int) error {
	if err := o.adminSvc.UpdateConfig(goal, manualAdj, note, overwrite, operSeq); err != nil {
		return err
	}
	if err := o.snapshotJob.CreateSnapshotNow(); err != nil {
		return fmt.Errorf("config saved but snapshot refresh failed: %w", err)
	}
	o.donationSvc.InvalidateCache()
	return nil
}

func (o *DonationConfigOrchestrator) GetHistory(days int) ([]model.DonationSnapshot, error) {
	return o.adminSvc.GetHistory(days)
}

func (o *DonationConfigOrchestrator) ListOrders(page, size int, search, status, payType string) ([]model.AdminDonationOrderRow, int, error) {
	return o.adminSvc.ListOrders(page, size, search, status, payType)
}

func (o *DonationConfigOrchestrator) UpdateOrder(seq int, payment string, amount int) error {
	return o.adminSvc.UpdateOrder(seq, payment, amount)
}
