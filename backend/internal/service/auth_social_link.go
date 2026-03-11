// auth_social_link.go — Social account linking: member lookup, creation, and social profile attachment
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

// SocialLinkParams holds the inputs for the social account linking flow.
type SocialLinkParams struct {
	Provider string
	SocialID string
	Email    string
	Name     string
	Phone    string
	FN       string
	Nick     string
	Dept     string
	JobCat   *int
	BizName  string
	BizDesc  string
	BizAddr  string
}

var ErrNameMismatch = errors.New("phone registered but name does not match")

// LinkSocialAccount finds or creates a member and attaches the social provider link.
// Returns the resolved user or an error (ErrNameMismatch for phone/name conflict).
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if existing.USRName != params.Name {
			return nil, ErrNameMismatch
		}
		if err := s.InsertSocialLink(existing.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
			return nil, err
		}
		return existing, nil
	}

	newUser, err := memberSvc.CreateMember(
		params.SocialID, params.Name, params.Phone, params.FN, params.Email,
		params.Nick, params.Dept, params.JobCat, params.BizName, params.BizDesc, params.BizAddr,
	)
	if err != nil {
		return nil, err
	}
	if err := s.InsertSocialLink(newUser.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
		return nil, err
	}
	return newUser, nil
}
