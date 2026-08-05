package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	ErrLoginSuspended = errors.New("account suspended")
	ErrLoginWithdrawn = errors.New("account withdrawn")
)

type LoginEligibilityPolicy struct{}

func (LoginEligibilityPolicy) EnsureLoginAllowed(user *model.User) error {
	if user == nil {
		return ErrLoginSuspended
	}
	return (LoginEligibilityPolicy{}).EnsureStatusAllowed(user.USRStatus)
}

func (LoginEligibilityPolicy) EnsureStatusAllowed(status string) error {
	switch status {
	case "BAA", "BBB", "CCC", "ZZZ":
		return nil
	case "AAA":
		return ErrLoginWithdrawn
	default:
		return ErrLoginSuspended
	}
}

func LoginErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrLoginWithdrawn):
		return "ACCOUNT_WITHDRAWN"
	case errors.Is(err, ErrLoginSuspended):
		return "ACCOUNT_SUSPENDED"
	default:
		return "LOGIN_NOT_ALLOWED"
	}
}
