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
	ExistingEmail       string
	ExistingPassword    string
	EncryptedCredential string
}

// ErrPhoneAlreadyRegistered is returned from mode=new when the phone belongs to another member.
var ErrPhoneAlreadyRegistered = errors.New("phone already registered to another member")

// ErrPhoneNotFound is returned from mode=merge when the phone no longer matches any member.
var ErrPhoneNotFound = errors.New("phone does not match any existing member")

var ErrExistingAccountReauthenticationRequired = errors.New("existing account reauthentication required")

var ErrAccountMergeNotSupported = errors.New("account merge not supported")

// LinkSocialAccount either creates a new member or merges the social account into an
// existing one, depending on params.Mode. The mode is chosen upstream by the caller,
// which has already confirmed with the user via the signup form.
// Returns the resolved user, whether a new member was created, or an error.
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	mode := params.Mode
	if mode == "" {
		mode = SocialLinkModeNew
	}
	requiresPhone := mode == SocialLinkModeNew || (mode == SocialLinkModeMerge && params.ExistingEmail == "")
	if requiresPhone {
		canonicalPhone := model.NormalizePhoneNumber(params.Phone)
		if !canonicalPhone.Valid() {
			return nil, false, ErrInvalidPhone
		}
		params.Phone = canonicalPhone.String()
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
	if (params.ExistingUSRID == "" && params.ExistingEmail == "") || params.ExistingPassword == "" {
		return nil, false, ErrExistingAccountReauthenticationRequired
	}
	var existing *model.User
	var err error
	if params.ExistingEmail != "" {
		existing, err = memberSvc.LoginWithEmailPassword(params.ExistingEmail, params.ExistingPassword)
	} else {
		existing, err = memberSvc.LoginWithPassword(params.ExistingUSRID, params.ExistingPassword)
	}
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, ErrExistingAccountReauthenticationRequired
	}
	if params.ExistingEmail != "" {
		user, attachErr := s.repo.AttachSocialAccount(repository.SocialAccountFields{
			Provider:            params.Provider,
			SocialID:            params.SocialID,
			SocialEmail:         params.Email,
			USRSeq:              existing.USRSeq,
			EncryptedCredential: params.EncryptedCredential,
		})
		if errors.Is(attachErr, repository.ErrSocialIdentityOwner) {
			return nil, false, ErrAccountMergeNotSupported
		}
		if attachErr != nil {
			return nil, false, attachErr
		}
		return user, false, nil
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
	if errors.Is(err, repository.ErrSocialIdentityOwner) {
		return nil, false, ErrAccountMergeNotSupported
	}
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
	if errors.Is(err, repository.ErrPhoneAlreadyClaimed) {
		return nil, false, ErrPhoneAlreadyRegistered
	}
	if errors.Is(err, repository.ErrInvalidPhone) {
		return nil, false, ErrInvalidPhone
	}
	if err != nil {
		return nil, false, err
	}
	return newUser, true, nil
}
