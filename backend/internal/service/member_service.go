// member_service.go — Member lookup, creation, and registration operations for authentication flows.
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// ErrPendingApproval is returned when a member exists but is awaiting admin approval (BBB status).
var ErrPendingApproval = errors.New("member pending approval")

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

// LoginWithPassword hashes the password and looks up an active member (status >= CCC).
// Returns ErrPendingApproval if credentials match but the member is still awaiting approval.
func (s *MemberService) LoginWithPassword(usrID, password string) (*model.User, error) {
	hashed := MysqlNativePassword(password)
	user, err := s.repo.FindMemberByLogin(usrID, hashed)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}
	// Check if credentials match a pending (BBB) account
	anyUser, err := s.repo.FindMemberByIDAndPwdAny(usrID, hashed)
	if err != nil {
		return nil, err
	}
	if anyUser != nil && anyUser.USRStatus == "BBB" {
		return nil, ErrPendingApproval
	}
	return nil, nil
}

// FindMemberByPhone finds an active member by phone number.
func (s *MemberService) FindMemberByPhone(phone string) (*model.User, error) {
	return s.repo.FindMemberByPhone(phone)
}

// CreateMember inserts a new member with USR_ID = "K" + kakaoID and returns the created user.
func (s *MemberService) CreateMember(kakaoID, name, phone, fn, email, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic string) (*model.User, error) {
	usrID := "K" + kakaoID
	usrSeq, err := s.repo.InsertMember(usrID, name, phone, fn, email, fmDept, jobCat, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic)
	if err != nil {
		return nil, err
	}
	return s.repo.GetMemberBySeq(usrSeq)
}
