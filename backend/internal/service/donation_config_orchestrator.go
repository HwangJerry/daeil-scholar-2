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
func (o *DonationConfigOrchestrator) UpdateConfig(goal int64, manualAdj int64, manualDonorCnt int, note string, overwrite bool, operSeq int) error {
	if err := o.adminSvc.UpdateConfig(goal, manualAdj, manualDonorCnt, note, overwrite, operSeq); err != nil {
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
	o.donationSvc.InvalidateCache()
	return order, nil
}

func (o *DonationConfigOrchestrator) UpdateOrder(seq int64, input model.DonationOrderInput, operSeq int, ip string) (*model.DonationOrder, error) {
	order, err := o.adminSvc.UpdateOrder(seq, input, operSeq, ip)
	if err != nil {
		return nil, err
	}
	o.donationSvc.InvalidateCache()
	return order, nil
}
