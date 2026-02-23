// ad_service.go — Business logic for ad retrieval and event tracking
package service

import "github.com/dflh-saf/backend/internal/repository"

// AdService handles ad retrieval and event tracking.
type AdService struct {
	repo *repository.AdRepository
}

// NewAdService creates a new AdService.
func NewAdService(repo *repository.AdRepository) *AdService {
	return &AdService{repo: repo}
}

// LogEvent records a view or click event for an ad (fire-and-forget).
func (s *AdService) LogEvent(maSeq int, usrSeq int, eventType string, ipAddr string) {
	_ = s.repo.LogAdEvent(maSeq, usrSeq, eventType, ipAddr)
}
