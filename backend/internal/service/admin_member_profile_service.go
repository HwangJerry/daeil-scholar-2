// admin_member_profile_service.go — Validation for admin member search filters and profile corrections
package service

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var (
	ErrInvalidMemberFilter = errors.New("invalid member filter")
	ErrMemberNameRequired  = errors.New("member name required")
	ErrInvalidMemberEmail  = errors.New("invalid member email")
	ErrInvalidMemberCohort = errors.New("invalid member cohort")
	ErrMemberNotFound      = repository.ErrMemberNotFound
)

// Column limits of the legacy WEO_MEMBER table.
const (
	memberNameMaxChars  = 100 // USR_NAME varchar(100)
	memberEmailMaxChars = 200 // USR_EMAIL varchar(200)
	memberCohortMax     = 99  // USR_FN int(2)
	filterDateLayout    = "2006-01-02"
)

func normalizeMemberFilter(filter model.AdminMemberFilter) (model.AdminMemberFilter, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.FN = strings.TrimSpace(filter.FN)
	filter.Dept = strings.TrimSpace(filter.Dept)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.RegFrom = strings.TrimSpace(filter.RegFrom)
	filter.RegTo = strings.TrimSpace(filter.RegTo)
	if filter.FN != "" && !isValidCohort(filter.FN) {
		return filter, ErrInvalidMemberFilter
	}
	if filter.Dept != "" && !model.IsValidDepartment(filter.Dept) {
		return filter, ErrInvalidMemberFilter
	}
	for _, date := range []string{filter.RegFrom, filter.RegTo} {
		if date == "" {
			continue
		}
		if _, err := time.Parse(filterDateLayout, date); err != nil {
			return filter, ErrInvalidMemberFilter
		}
	}
	if filter.RegFrom != "" && filter.RegTo != "" && filter.RegFrom > filter.RegTo {
		return filter, ErrInvalidMemberFilter
	}
	return filter, nil
}

func isValidCohort(value string) bool {
	cohort, err := strconv.Atoi(value)
	return err == nil && cohort >= 1 && cohort <= memberCohortMax
}

// UpdateProfile validates and applies an administrator's correction of a
// member's name, phone, contact email, cohort and department.
func (s *AdminMemberService) UpdateProfile(seq int, req model.AdminMemberProfileUpdate) error {
	req, err := normalizeMemberProfile(req)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateMemberProfile(seq, req); err != nil {
		switch {
		case errors.Is(err, repository.ErrPhoneAlreadyClaimed):
			return ErrPhoneTaken
		case errors.Is(err, repository.ErrInvalidPhone):
			return ErrInvalidPhone
		}
		return err
	}
	return nil
}

func normalizeMemberProfile(req model.AdminMemberProfileUpdate) (model.AdminMemberProfileUpdate, error) {
	req.USRName = strings.TrimSpace(req.USRName)
	req.USREmail = strings.TrimSpace(req.USREmail)
	req.USRFN = strings.TrimSpace(req.USRFN)
	req.USRDept = strings.TrimSpace(req.USRDept)
	req.USRPhone = strings.TrimSpace(req.USRPhone)

	if req.USRName == "" || utf8.RuneCountInString(req.USRName) > memberNameMaxChars {
		return req, ErrMemberNameRequired
	}
	if req.USREmail != "" {
		address, err := mail.ParseAddress(req.USREmail)
		if err != nil || address.Address != req.USREmail || utf8.RuneCountInString(req.USREmail) > memberEmailMaxChars {
			return req, ErrInvalidMemberEmail
		}
	}
	if req.USRFN != "" && !isValidCohort(req.USRFN) {
		return req, ErrInvalidMemberCohort
	}
	if req.USRDept != "" && !model.IsValidDepartment(req.USRDept) {
		return req, ErrInvalidDepartment
	}
	if req.USRPhone != "" {
		phone := model.NormalizePhoneNumber(req.USRPhone)
		if !phone.Valid() {
			return req, ErrInvalidPhone
		}
		req.USRPhone = phone.String()
	}
	return req, nil
}
