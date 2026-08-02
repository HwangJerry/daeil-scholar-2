// member_service.go — Member lookup, creation, and registration operations for authentication flows.
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// ErrPendingApproval remains as a compatibility alias for older handlers.
var ErrPendingApproval = ErrLoginPending

// ErrIDTaken is returned when the requested user ID is already in use.
var ErrIDTaken = errors.New("user ID already taken")

// ErrPhoneTaken is returned when the phone number is already registered.
var ErrPhoneTaken = errors.New("phone number already taken")

// ErrEmailTaken is returned when the email address is already registered.
var ErrEmailTaken = errors.New("email already taken")

// MemberService handles member lookup and creation, used by auth handlers.
type MemberService struct {
	repo *repository.AuthRepository
}

// NewMemberService creates a MemberService backed by the given repository.
func NewMemberService(repo *repository.AuthRepository) *MemberService {
	return &MemberService{repo: repo}
}

// LoginWithPassword verifies credentials first, then applies the same status
// policy used by social login and refresh.
func (s *MemberService) LoginWithPassword(usrID, password string) (*model.User, error) {
	hashed := MysqlNativePassword(password)
	user, err := s.repo.FindMemberByIDAndPwdAny(usrID, hashed)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	return user, nil
}

// LoginWithEmailPassword verifies a canonical email credential while keeping
// alumni verification separate from account lifecycle eligibility.
func (s *MemberService) LoginWithEmailPassword(email, password string) (*model.User, error) {
	hashed := MysqlNativePassword(password)
	user, err := s.repo.FindMemberByEmailAndPwdAny(email, hashed)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	if err := (LoginEligibilityPolicy{}).EnsureLoginAllowed(user); err != nil {
		return nil, err
	}
	return user, nil
}

// FindMemberByPhone finds an active member by phone number.
func (s *MemberService) FindMemberByPhone(phone string) (*model.User, error) {
	return s.repo.FindMemberByPhone(model.NormalizePhoneNumber(phone).String())
}

// CreateMember inserts a new member with USR_ID = "K" + kakaoID and returns the created user.
// profileImageURL seeds USR_PHOTO when provided by the social provider (Kakao profile_image_url);
// pass an empty string when consent was declined.
func (s *MemberService) CreateMember(kakaoID, name, phone, fn, email, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic, profileImageURL string) (*model.User, error) {
	return s.CreateSocialMember(
		model.SocialProviderKakao,
		kakaoID,
		name,
		phone,
		fn,
		email,
		fmDept,
		jobCat,
		bizName,
		bizDesc,
		bizAddr,
		position,
		usrPhonePublic,
		usrEmailPublic,
		profileImageURL,
	)
}

func (s *MemberService) CreateSocialMember(provider model.SocialProvider, subject, name, phone, fn, email, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic, profileImageURL string) (*model.User, error) {
	usrID := socialMemberID(provider, subject)
	usrSeq, err := s.repo.InsertMember(
		usrID,
		name,
		model.NormalizePhoneNumber(phone).String(),
		fn,
		email,
		fmDept,
		jobCat,
		bizName,
		bizDesc,
		bizAddr,
		position,
		usrPhonePublic,
		usrEmailPublic,
		profileImageURL,
	)
	if err != nil {
		return nil, err
	}
	return s.repo.GetMemberBySeq(usrSeq)
}

func socialMemberID(provider model.SocialProvider, subject string) string {
	if provider == model.SocialProviderKakao {
		return "K" + subject
	}
	digest := sha256.Sum256([]byte(string(provider) + ":" + subject))
	return "S" + string(provider) + hex.EncodeToString(digest[:])[:29]
}
