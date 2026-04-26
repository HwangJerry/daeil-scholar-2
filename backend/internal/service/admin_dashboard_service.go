// AdminDashboardService — orchestrates domain services to assemble dashboard stats
package service

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type AdminDashboardService struct {
	memberSvc   *AdminMemberService
	noticeSvc   *AdminNoticeService
	adSvc       *AdminAdService
	donationSvc *DonationService
	visitSvc    *VisitService
}

func NewAdminDashboardService(
	memberSvc *AdminMemberService,
	noticeSvc *AdminNoticeService,
	adSvc *AdminAdService,
	donationSvc *DonationService,
	visitSvc *VisitService,
) *AdminDashboardService {
	return &AdminDashboardService{
		memberSvc:   memberSvc,
		noticeSvc:   noticeSvc,
		adSvc:       adSvc,
		donationSvc: donationSvc,
		visitSvc:    visitSvc,
	}
}

// GetStats assembles pre-computed stats from each domain service.
func (s *AdminDashboardService) GetStats() (*model.DashboardStats, error) {
	memberStats, _ := s.memberSvc.GetMemberStats()
	totalNotices, _ := s.noticeSvc.CountNotices()
	donation, _ := s.donationSvc.GetSummary()
	adStats, _ := s.adSvc.GetDashboardAdStats()
	dauToday, mauCurrent, _ := s.visitSvc.DashboardCounts()

	if memberStats == nil {
		memberStats = &MemberStats{}
	}
	if donation == nil {
		donation = &model.DonationSummary{}
	}
	if adStats == nil {
		adStats = &model.DashboardAdStats{}
	}

	return &model.DashboardStats{
		TotalMembers:       memberStats.TotalMembers,
		KakaoLinkedMembers: memberStats.KakaoLinkedMembers,
		RecentLoginCount:   memberStats.RecentLoginCount,
		PendingApprovals:   memberStats.PendingApprovals,
		TotalNotices:       totalNotices,
		DAUToday:           dauToday,
		MAUCurrent:         mauCurrent,
		Donation:           *donation,
		AdStats:            *adStats,
	}, nil
}

// ActiveUsers returns the DAU/MAU time series for the admin analytics endpoint.
func (s *AdminDashboardService) ActiveUsers(from, to time.Time) (*model.ActiveUsersResponse, error) {
	return s.visitSvc.ActiveUsers(from, to)
}
