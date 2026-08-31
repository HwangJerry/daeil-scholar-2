// auth_social_link.go — Social account linking: member lookup, creation, and social profile attachment
package service

import (
	"errors"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// SocialLinkMode selects the behavior of LinkSocialAccount.
type SocialLinkMode string

const (
	SocialLinkModeNew SocialLinkMode = "new"
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
	EncryptedCredential string
}

var (
	// ErrPhoneAlreadyRegistered is returned from mode=new when the phone belongs to another member.
	ErrPhoneAlreadyRegistered = errors.New("phone already registered to another member")
	// ErrOwnershipConfirmationRequired is returned when an existing member has the same phone and name.
	ErrOwnershipConfirmationRequired = errors.New("ownership confirmation required")
	// ErrSocialAccountAlreadyLinked is returned when the social identity belongs to another member.
	ErrSocialAccountAlreadyLinked = errors.New("social account already linked to another member")
)

// LinkSocialAccount creates a new member from a verified social identity.
// Returns the created user, whether a new member was created, or an error.
func (s *AuthService) LinkSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	mode := params.Mode
	if mode == "" {
		mode = SocialLinkModeNew
	}
	if mode != SocialLinkModeNew {
		return nil, false, errors.New("invalid social link mode: " + string(mode))
	}
	canonicalPhone := model.NormalizePhoneNumber(params.Phone)
	if !canonicalPhone.Valid() {
		return nil, false, ErrInvalidPhone
	}
	params.Phone = canonicalPhone.String()
	return s.createNewSocialAccount(params, memberSvc)
}

func (s *AuthService) createNewSocialAccount(params SocialLinkParams, memberSvc *MemberService) (*model.User, bool, error) {
	existing, err := memberSvc.FindMemberByPhone(params.Phone)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		existingName := strings.Join(strings.Fields(existing.USRName), "")
		requestedName := strings.Join(strings.Fields(params.Name), "")
		if existingName == requestedName {
			return nil, false, ErrOwnershipConfirmationRequired
		}
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
	if errors.Is(err, repository.ErrSocialIdentityAlreadyLinked) {
		return nil, false, ErrSocialAccountAlreadyLinked
	}
	if errors.Is(err, repository.ErrInvalidPhone) {
		return nil, false, ErrInvalidPhone
	}
	if err != nil {
		return nil, false, err
	}
	return newUser, true, nil
}
