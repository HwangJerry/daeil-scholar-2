// profile_service.go — Business logic for user profile management
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type ProfileService struct {
	repo *repository.ProfileRepository
}

func NewProfileService(repo *repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetProfile(usrSeq int) (*model.UserProfile, error) {
	return s.repo.GetProfile(usrSeq)
}

func (s *ProfileService) UpdateProfile(usrSeq int, req model.ProfileUpdateRequest) error {
	if req.Tags != nil {
		if err := ValidateTags(req.Tags); err != nil {
			return err
		}
	}
	if err := s.repo.UpdateProfile(usrSeq, req); err != nil {
		return err
	}
	if req.Tags != nil {
		return s.repo.SaveUserTags(usrSeq, req.Tags)
	}
	return nil
}

