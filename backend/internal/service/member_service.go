// member_service.go — Member lookup and creation operations for authentication flows.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// MemberService handles member lookup and creation, used by auth handlers.
type MemberService struct {
	repo *repository.AuthRepository
}

// NewMemberService creates a MemberService backed by the given repository.
func NewMemberService(repo *repository.AuthRepository) *MemberService {
	return &MemberService{repo: repo}
}

// LoginWithPassword hashes the password using MySQL native format and looks up the member.
func (s *MemberService) LoginWithPassword(usrID, password string) (*model.User, error) {
	hashed := MysqlNativePassword(password)
	return s.repo.FindMemberByLogin(usrID, hashed)
}

// FindMemberByPhone finds an active member by phone number.
func (s *MemberService) FindMemberByPhone(phone string) (*model.User, error) {
	return s.repo.FindMemberByPhone(phone)
}

// CreateMember inserts a new member with USR_ID = "K" + kakaoID and returns the created user.
func (s *MemberService) CreateMember(kakaoID, name, phone, fn, email string) (*model.User, error) {
	usrID := "K" + kakaoID
	usrSeq, err := s.repo.InsertMember(usrID, name, phone, fn, email)
	if err != nil {
		return nil, err
	}
	return s.repo.GetMemberBySeq(usrSeq)
}
