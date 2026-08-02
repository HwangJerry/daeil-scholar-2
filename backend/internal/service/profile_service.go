// profile_service.go — Business logic for user profile management
package service

import (
	"errors"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var ErrInvalidDepartment = errors.New("invalid department")

var ErrAcademicInformationRequired = errors.New("complete academic information is required")

type ProfileService struct {
	repo *repository.ProfileRepository
}

func NewProfileService(repo *repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetProfile(usrSeq int) (*model.UserProfile, error) {
	return s.repo.GetProfile(usrSeq)
}

func (s *ProfileService) GetAlumniVerification(usrSeq int) (*model.AlumniVerification, error) {
	return s.repo.GetAlumniVerification(usrSeq)
}

func (s *ProfileService) SubmitAlumniVerification(usrSeq int, req model.AlumniVerificationSubmissionRequest) error {
	if req.GraduationYear <= 0 || strings.TrimSpace(req.Cohort) == "" || strings.TrimSpace(req.Department) == "" {
		return ErrAcademicInformationRequired
	}
	req.Cohort = strings.TrimSpace(req.Cohort)
	req.Department = strings.TrimSpace(req.Department)
	if !model.IsValidDepartment(req.Department) {
		return ErrInvalidDepartment
	}
	return s.repo.SubmitAlumniVerification(usrSeq, req)
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
