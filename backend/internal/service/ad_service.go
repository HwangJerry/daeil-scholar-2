package service

import "github.com/dflh-saf/backend/internal/repository"

type AdService struct {
	repo *repository.AdRepository
}

func NewAdService(repo *repository.AdRepository) *AdService {
	return &AdService{repo: repo}
}

func (s *AdService) LogEvent(maSeq int, usrSeq int, eventType string, ipAddr string) {
	_ = s.repo.LogAdEvent(maSeq, usrSeq, eventType, ipAddr)
}
