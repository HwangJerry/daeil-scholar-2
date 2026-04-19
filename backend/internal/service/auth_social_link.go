// auth_social_link.go — Social account linking: member lookup, creation, and social profile attachment
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

// SocialLinkParams holds the inputs for the social account linking flow.
type SocialLinkParams struct {
	Provider       string
	SocialID       string
	Email          string
	Name           string
	Phone          string
	FN             string
	FmDept         string
	JobCat         *int
	BizName        string
	BizDesc        string
	BizAddr        string
	Position       string
	Tags           []string
	USRPhonePublic string
	USREmailPublic string
}

var ErrNameMismatch = errors.New("phone registered but name does not match")

// LinkSocialAccount finds or creates a member and attaches the social provider link.
// Returns the resolved user, whether a new member was created, or an error.
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, false, err
	}

	if existing != nil {
		if existing.USRName != params.Name {
			return nil, false, ErrNameMismatch
		}
		if err := s.InsertSocialLink(existing.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}

	newUser, err := memberSvc.CreateMember(
		params.SocialID, params.Name, params.Phone, params.FN, params.Email,
		params.FmDept, params.JobCat, params.BizName, params.BizDesc, params.BizAddr,
		params.Position, params.USRPhonePublic, params.USREmailPublic,
	)
	if err != nil {
		return nil, false, err
	}
	if err := s.InsertSocialLink(newUser.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
		return nil, false, err
	}
	return newUser, true, nil
}
