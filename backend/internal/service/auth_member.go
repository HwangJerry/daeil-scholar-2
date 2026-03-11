// auth_member.go — Member lookup and social account linking
package service

import "github.com/dflh-saf/backend/internal/model"

// FindMemberBySocialID looks up a member by their linked social provider and ID.
func (s *AuthService) FindMemberBySocialID(gate string, socialID string) (*model.User, error) {
	return s.repo.FindMemberBySocialID(gate, socialID)
}

// FindMemberByKakaoID looks up a member by their linked Kakao social ID.
func (s *AuthService) FindMemberByKakaoID(kakaoID string) (*model.User, error) {
	return s.FindMemberBySocialID("KT", kakaoID)
}

// FindMemberByNamePhone looks up a member by name and phone number.
func (s *AuthService) FindMemberByNamePhone(name string, phone string) (*model.User, error) {
	return s.repo.FindMemberByNamePhone(name, phone)
}

// FindMemberByFNName looks up a member by alumni class identifier and name.
func (s *AuthService) FindMemberByFNName(fn string, name string) (*model.User, error) {
	return s.repo.FindMemberByFNName(fn, name)
}

// InsertSocialLink creates a social login link record for a member.
func (s *AuthService) InsertSocialLink(usrSeq int, gate string, socialID string, email string) error {
	return s.repo.InsertSocialLink(usrSeq, gate, socialID, email)
}
