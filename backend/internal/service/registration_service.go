// registration_service.go — Orchestrates new member signup: member creation + initial tag setup.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// RegistrationService coordinates the full signup flow: member creation (AuthRepository)
// and optional initial tag setup (ProfileRepository). Each repository retains its own
// single responsibility; this service owns only the signup orchestration.
type RegistrationService struct {
	memberRepo  *repository.AuthRepository
	profileRepo *repository.ProfileRepository
}

// NewRegistrationService creates a RegistrationService.
func NewRegistrationService(memberRepo *repository.AuthRepository, profileRepo *repository.ProfileRepository) *RegistrationService {
	return &RegistrationService{memberRepo: memberRepo, profileRepo: profileRepo}
}

// IsIDAvailable returns true if the given user ID is not yet taken.
func (s *RegistrationService) IsIDAvailable(usrID string) (bool, error) {
	exists, err := s.memberRepo.CheckIDExists(usrID)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// IsPhoneAvailable returns true if the phone number is not yet registered.
func (s *RegistrationService) IsPhoneAvailable(phone string) (bool, error) {
	exists, err := s.memberRepo.CheckPhoneExists(phone)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// IsEmailAvailable returns true if the email is not yet registered.
func (s *RegistrationService) IsEmailAvailable(email string) (bool, error) {
	exists, err := s.memberRepo.CheckEmailExists(email)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// SaveInitialTags persists tags for a newly created member. Used by the social link flow.
// Returns ErrTagContainsWhitespace if any tag contains whitespace.
func (s *RegistrationService) SaveInitialTags(usrSeq int, tags []string) error {
	if err := ValidateTags(tags); err != nil {
		return err
	}
	return s.profileRepo.SaveUserTags(usrSeq, tags)
}

// Register validates uniqueness, creates the member, and persists any signup-time tags.
// Returns the created user or ErrIDTaken / ErrPhoneTaken on conflict.
func (s *RegistrationService) Register(req model.RegisterRequest) (*model.User, error) {
	idExists, err := s.memberRepo.CheckIDExists(req.UsrID)
	if err != nil {
		return nil, err
	}
	if idExists {
		return nil, ErrIDTaken
	}

	phoneExists, err := s.memberRepo.CheckPhoneExists(req.Phone)
	if err != nil {
		return nil, err
	}
	if phoneExists {
		return nil, ErrPhoneTaken
	}

	if req.Email != "" {
		emailExists, err := s.memberRepo.CheckEmailExists(req.Email)
		if err != nil {
			return nil, err
		}
		if emailExists {
			return nil, ErrEmailTaken
		}
	}

	if len(req.Tags) > 0 {
		if err := ValidateTags(req.Tags); err != nil {
			return nil, err
		}
	}

	hashed := MysqlNativePassword(req.Password)
	usrSeq, err := s.memberRepo.InsertMemberWithPwd(req, hashed)
	if err != nil {
		return nil, err
	}

	if len(req.Tags) > 0 {
		if err := s.profileRepo.SaveUserTags(usrSeq, req.Tags); err != nil {
			return nil, err
		}
	}

	return s.memberRepo.GetMemberBySeq(usrSeq)
}
