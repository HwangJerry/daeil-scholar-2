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

// FindMemberByEmail looks up an active member by email address.
func (s *AuthService) FindMemberByEmail(email string) (*model.User, error) {
	return s.repo.FindMemberByEmail(email)
}

// FindMemberByFNName looks up a member by alumni class identifier and name.
func (s *AuthService) FindMemberByFNName(fn string, name string) (*model.User, error) {
	return s.repo.FindMemberByFNName(fn, name)
}

// InsertSocialLink creates a social login link record for a member.
func (s *AuthService) InsertSocialLink(usrSeq int, gate string, socialID string, email string) error {
	return s.repo.InsertSocialLink(usrSeq, gate, socialID, email)
}

// UpdateProfilePhotoIfEmpty sets USR_PHOTO only when empty, used by auto-link to seed avatars without overwriting.
func (s *AuthService) UpdateProfilePhotoIfEmpty(usrSeq int, url string) error {
	return s.repo.UpdateProfilePhotoIfEmpty(usrSeq, url)
}

// UpdateMemberOptionalFields updates the merge-editable optional fields on an existing member.
func (s *AuthService) UpdateMemberOptionalFields(usrSeq int, fn, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic string) error {
	return s.repo.UpdateMemberOptionalFields(usrSeq, fn, fmDept, jobCat, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic)
}
