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
	memberRepo  registrationMemberRepository
	profileRepo registrationProfileRepository
	credentials passwordCredentialManager
	signupRepo  passwordSignupRepository
	hasher      *PasswordHasher
}

type registrationMemberRepository interface {
	CheckIDExists(string) (bool, error)
	CheckPhoneExists(string) (bool, error)
	CheckEmailExists(string) (bool, error)
	InsertMemberWithPwd(model.RegisterRequest, string) (int, error)
	GetMemberBySeq(int) (*model.User, error)
}

type registrationProfileRepository interface {
	SaveUserTags(int, []string) error
}

type passwordSignupRepository interface {
	CreatePasswordAccount(model.RegisterRequest, string, model.PasswordCredential) (int, error)
}

// NewRegistrationService creates a RegistrationService.
func NewRegistrationService(memberRepo *repository.AuthRepository, profileRepo *repository.ProfileRepository) *RegistrationService {
	return &RegistrationService{memberRepo: memberRepo, profileRepo: profileRepo}
}

func NewRegistrationServiceWithPasswordCredentials(memberRepo registrationMemberRepository, profileRepo registrationProfileRepository, credentials passwordCredentialManager) *RegistrationService {
	return &RegistrationService{memberRepo: memberRepo, profileRepo: profileRepo, credentials: credentials}
}

func NewTransactionalRegistrationService(memberRepo registrationMemberRepository, profileRepo registrationProfileRepository, signupRepo passwordSignupRepository) *RegistrationService {
	return &RegistrationService{memberRepo: memberRepo, profileRepo: profileRepo, signupRepo: signupRepo, hasher: NewPasswordHasher()}
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
	exists, err := s.memberRepo.CheckPhoneExists(model.NormalizePhoneNumber(phone).String())
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
	req.Phone = model.NormalizePhoneNumber(req.Phone).String()
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
	if s.signupRepo != nil {
		credential, err := s.newSignupPasswordCredential(req.Password)
		if err != nil {
			return nil, err
		}
		usrSeq, err := s.signupRepo.CreatePasswordAccount(req, hashed, credential)
		if err != nil {
			return nil, err
		}
		return s.memberRepo.GetMemberBySeq(usrSeq)
	}
	usrSeq, err := s.memberRepo.InsertMemberWithPwd(req, hashed)
	if err != nil {
		return nil, err
	}
	if s.credentials != nil {
		if err := s.credentials.StoreIdentityPassword(model.IdentityProviderLocalUsername, req.UsrID, usrSeq, req.Password); err != nil {
			return nil, err
		}
	}

	if len(req.Tags) > 0 {
		if err := s.profileRepo.SaveUserTags(usrSeq, req.Tags); err != nil {
			return nil, err
		}
	}

	return s.memberRepo.GetMemberBySeq(usrSeq)
}

func (s *RegistrationService) newSignupPasswordCredential(password string) (model.PasswordCredential, error) {
	return s.hasher.NewCredential(0, model.IdentityProviderLocalUsername, password)
}
