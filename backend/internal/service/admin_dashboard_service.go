// AdminDashboardService — orchestrates domain services to assemble dashboard stats
package service

import "github.com/dflh-saf/backend/internal/model"

type AdminDashboardService struct {
	memberSvc   *AdminMemberService
	noticeSvc   *AdminNoticeService
	adSvc       *AdminAdService
	donationSvc *DonationService
}

func NewAdminDashboardService(
	memberSvc *AdminMemberService,
	noticeSvc *AdminNoticeService,
	adSvc *AdminAdService,
	donationSvc *DonationService,
) *AdminDashboardService {
	return &AdminDashboardService{
		memberSvc:   memberSvc,
		noticeSvc:   noticeSvc,
		adSvc:       adSvc,
		donationSvc: donationSvc,
	}
}

// GetStats assembles pre-computed stats from each domain service.
func (s *AdminDashboardService) GetStats() (*model.DashboardStats, error) {
	memberStats, _ := s.memberSvc.GetMemberStats()
	totalNotices, _ := s.noticeSvc.CountNotices()
	donation, _ := s.donationSvc.GetSummary()
	adStats, _ := s.adSvc.GetDashboardAdStats()

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
		TotalNotices:       totalNotices,
		Donation:           *donation,
		AdStats:            *adStats,
	}, nil
}
