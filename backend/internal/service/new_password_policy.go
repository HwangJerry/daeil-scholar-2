package service

import (
	"errors"
	"regexp"
)

const MinNewPasswordUTF8Bytes = 8

var ErrInvalidNewPassword = errors.New("invalid new password")

var (
	newPasswordLetter  = regexp.MustCompile(`[a-zA-Z]`)
	newPasswordNumber  = regexp.MustCompile(`[0-9]`)
	newPasswordSpecial = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// ValidateNewPassword deliberately preserves the existing change/reset contract:
// UTF-8 bytes, ASCII letters/digits, and any non-ASCII-alphanumeric character.
// Do not apply this policy to legacy login or credential hash transitions.
func ValidateNewPassword(password string) error {
	if len(password) < MinNewPasswordUTF8Bytes {
		return newPasswordPolicyError("비밀번호는 최소 8자 이상이어야 합니다")
	}
	if !newPasswordLetter.MatchString(password) || !newPasswordNumber.MatchString(password) || !newPasswordSpecial.MatchString(password) {
		return newPasswordPolicyError("비밀번호는 영문, 숫자, 특수문자를 모두 포함해야 합니다")
	}
	return nil
}

type newPasswordPolicyError string

func (e newPasswordPolicyError) Error() string        { return string(e) }
func (e newPasswordPolicyError) Is(target error) bool { return target == ErrInvalidNewPassword }
