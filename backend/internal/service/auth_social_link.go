// auth_social_link.go — Social account linking: member lookup, creation, and social profile attachment
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

// SocialLinkMode selects the behavior of LinkSocialAccount.
//
// - "new":   creates a fresh WEO_MEMBER row. Fails if the given phone already matches an existing row.
// - "merge": attaches the social link to an existing WEO_MEMBER row found by phone. Fails if no match.
type SocialLinkMode string

const (
	SocialLinkModeNew   SocialLinkMode = "new"
	SocialLinkModeMerge SocialLinkMode = "merge"
)

// SocialLinkParams holds the inputs for the social account linking flow.
type SocialLinkParams struct {
	Mode            SocialLinkMode
	Provider        string
	SocialID        string
	Email           string
	Name            string
	Phone           string
	FN              string
	FmDept          string
	JobCat          *int
	BizName         string
	BizDesc         string
	BizAddr         string
	Position        string
	Tags            []string
	USRPhonePublic  string
	USREmailPublic  string
	ProfileImageURL string
}

// ErrPhoneAlreadyRegistered is returned from mode=new when the phone belongs to another member.
var ErrPhoneAlreadyRegistered = errors.New("phone already registered to another member")

// ErrPhoneNotFound is returned from mode=merge when the phone no longer matches any member.
var ErrPhoneNotFound = errors.New("phone does not match any existing member")

// LinkSocialAccount either creates a new member or merges the social account into an
// existing one, depending on params.Mode. The mode is chosen upstream by the caller,
// which has already confirmed with the user via the signup form (new) or a match banner (merge).
// Returns the resolved user, whether a new member was created, or an error.
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	mode := params.Mode
	if mode == "" {
		mode = SocialLinkModeNew
	}

	switch mode {
	case SocialLinkModeMerge:
		return s.mergeSocialAccount(params, memberSvc)
	case SocialLinkModeNew:
		return s.createNewSocialAccount(params, memberSvc)
	default:
		return nil, false, errors.New("invalid social link mode: " + string(mode))
	}
}

func (s *AuthService) mergeSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, ErrPhoneNotFound
	}
	if err := s.repo.UpdateMemberOptionalFields(
		existing.USRSeq, params.FN, params.FmDept, params.JobCat,
		params.BizName, params.BizDesc, params.BizAddr, params.Position,
		params.USRPhonePublic, params.USREmailPublic,
	); err != nil {
		return nil, false, err
	}
	if err := s.InsertSocialLink(existing.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
		return nil, false, err
	}
	if params.ProfileImageURL != "" {
		if err := s.UpdateProfilePhotoIfEmpty(existing.USRSeq, params.ProfileImageURL); err != nil {
			return nil, false, err
		}
	}
	return existing, false, nil
}

func (s *AuthService) createNewSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return nil, false, ErrPhoneAlreadyRegistered
	}
	newUser, err := memberSvc.CreateMember(
		params.SocialID, params.Name, params.Phone, params.FN, params.Email,
		params.FmDept, params.JobCat, params.BizName, params.BizDesc, params.BizAddr,
		params.Position, params.USRPhonePublic, params.USREmailPublic, params.ProfileImageURL,
	)
	if err != nil {
		return nil, false, err
	}
	if err := s.InsertSocialLink(newUser.USRSeq, params.Provider, params.SocialID, params.Email); err != nil {
		return nil, false, err
	}
	return newUser, true, nil
}
