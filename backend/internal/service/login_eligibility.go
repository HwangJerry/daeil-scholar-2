package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	ErrLoginPending   = errors.New("account pending approval")
	ErrLoginSuspended = errors.New("account suspended")
	ErrLoginWithdrawn = errors.New("account withdrawn")
)

type LoginEligibilityPolicy struct{}

func (LoginEligibilityPolicy) EnsureLoginAllowed(user *model.User) error {
	if user == nil {
		return ErrLoginSuspended
	}
	switch user.USRStatus {
	case "CCC", "ZZZ":
		return nil
	case "BBB":
		return ErrLoginPending
	case "AAA":
		return ErrLoginWithdrawn
	default:
		return ErrLoginSuspended
	}
}

func LoginErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrLoginPending):
		return "ACCOUNT_PENDING"
	case errors.Is(err, ErrLoginWithdrawn):
		return "ACCOUNT_WITHDRAWN"
	case errors.Is(err, ErrLoginSuspended):
		return "ACCOUNT_SUSPENDED"
	default:
		return "LOGIN_NOT_ALLOWED"
	}
}
