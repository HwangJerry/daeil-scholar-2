// auth_social_link.go — Social account linking: member lookup, creation, and social profile attachment
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// SocialLinkMode selects the behavior of LinkSocialAccount.
//
// - "new":   creates a fresh WEO_MEMBER row. Fails if the given phone already matches an existing row.
// - "merge": reauthenticates an existing account and verifies the submitted phone before linking.
type SocialLinkMode string

const (
	SocialLinkModeNew   SocialLinkMode = "new"
	SocialLinkModeMerge SocialLinkMode = "merge"
)

// SocialLinkParams holds the inputs for the social account linking flow.
type SocialLinkParams struct {
	Mode                SocialLinkMode
	Provider            string
	SocialID            string
	Email               string
	Name                string
	Phone               string
	FN                  string
	FmDept              string
	JobCat              *int
	BizName             string
	BizDesc             string
	BizAddr             string
	Position            string
	Tags                []string
	USRPhonePublic      string
	USREmailPublic      string
	ProfileImageURL     string
	ExistingUSRID       string
	ExistingPassword    string
	EncryptedCredential string
}

// ErrPhoneAlreadyRegistered is returned from mode=new when the phone belongs to another member.
var ErrPhoneAlreadyRegistered = errors.New("phone already registered to another member")

// ErrPhoneNotFound is returned from mode=merge when the phone no longer matches any member.
var ErrPhoneNotFound = errors.New("phone does not match any existing member")

var ErrExistingAccountReauthenticationRequired = errors.New("existing account reauthentication required")

// LinkSocialAccount either creates a new member or merges the social account into an
// existing one, depending on params.Mode. The mode is chosen upstream by the caller,
// which has already confirmed with the user via the signup form.
// Returns the resolved user, whether a new member was created, or an error.
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	params.Phone = model.NormalizePhoneNumber(params.Phone).String()
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
	if params.ExistingUSRID == "" || params.ExistingPassword == "" {
		return nil, false, ErrExistingAccountReauthenticationRequired
	}
	existing, err := memberSvc.LoginWithPassword(params.ExistingUSRID, params.ExistingPassword)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, ErrExistingAccountReauthenticationRequired
	}
	if !existing.USRPhone.Valid ||
		model.NormalizePhoneNumber(existing.USRPhone.String) != model.NormalizePhoneNumber(params.Phone) {
		return nil, false, ErrPhoneNotFound
	}
	user, err := s.repo.MergeSocialAccount(repository.SocialAccountFields{
		Provider:            params.Provider,
		SocialID:            params.SocialID,
		SocialEmail:         params.Email,
		USRSeq:              existing.USRSeq,
		Name:                params.Name,
		Phone:               params.Phone,
		FN:                  params.FN,
		Email:               params.Email,
		FmDept:              params.FmDept,
		JobCat:              params.JobCat,
		BizName:             params.BizName,
		BizDesc:             params.BizDesc,
		BizAddr:             params.BizAddr,
		Position:            params.Position,
		USRPhonePublic:      params.USRPhonePublic,
		USREmailPublic:      params.USREmailPublic,
		ProfileImageURL:     params.ProfileImageURL,
		EncryptedCredential: params.EncryptedCredential,
	})
	if err != nil {
		return nil, false, err
	}
	return user, false, nil
}

func (s *AuthService) createNewSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return nil, false, ErrPhoneAlreadyRegistered
	}
	newUser, err := s.repo.CreateSocialAccount(repository.SocialAccountFields{
		Provider:            params.Provider,
		SocialID:            params.SocialID,
		SocialEmail:         params.Email,
		USRID:               socialMemberID(model.SocialProvider(params.Provider), params.SocialID),
		Name:                params.Name,
		Phone:               params.Phone,
		FN:                  params.FN,
		Email:               params.Email,
		FmDept:              params.FmDept,
		JobCat:              params.JobCat,
		BizName:             params.BizName,
		BizDesc:             params.BizDesc,
		BizAddr:             params.BizAddr,
		Position:            params.Position,
		USRPhonePublic:      params.USRPhonePublic,
		USREmailPublic:      params.USREmailPublic,
		ProfileImageURL:     params.ProfileImageURL,
		EncryptedCredential: params.EncryptedCredential,
	})
	if err != nil {
		return nil, false, err
	}
	return newUser, true, nil
}
