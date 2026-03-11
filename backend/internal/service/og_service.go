// og_service.go — Service for retrieving Open Graph metadata for social sharing bots
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// OGService retrieves minimal OG metadata for bot-facing HTML rendering.
type OGService struct {
	repo *repository.FeedRepository
}

// NewOGService creates a new OGService.
func NewOGService(repo *repository.FeedRepository) *OGService {
	return &OGService{repo: repo}
}

// GetOGData returns minimal OG data for a post (no hit increment).
func (s *OGService) GetOGData(seq int) (*model.OGData, error) {
	return s.repo.GetOGFields(seq)
}
