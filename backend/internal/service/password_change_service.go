// password_change_service.go — Business logic for authenticated password changes (id/pw users).
package service

import (
	"errors"
	"regexp"

	"github.com/dflh-saf/backend/internal/repository"
)

var (
	pwChangeHasLetter  = regexp.MustCompile(`[a-zA-Z]`)
	pwChangeHasNumber  = regexp.MustCompile(`[0-9]`)
	pwChangeHasSpecial = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

const minChangePwLength = 8

// PasswordChangeService handles password mutation for logged-in id/pw users.
// It is distinct from PasswordResetService, which handles the forgotten-password flow.
type PasswordChangeService struct {
	repo *repository.ProfileRepository
}

func NewPasswordChangeService(repo *repository.ProfileRepository) *PasswordChangeService {
	return &PasswordChangeService{repo: repo}
}

// ChangePassword verifies currentPwd against the stored hash and replaces it with newPwd.
// Returns errors.New("NO_PASSWORD") for Kakao-only users and "WRONG_PASSWORD" on mismatch.
func (s *PasswordChangeService) ChangePassword(usrSeq int, currentPwd, newPwd string) error {
	stored, err := s.repo.GetPasswordHash(usrSeq)
	if err != nil {
		return err
	}
	if stored == "" {
		return errors.New("NO_PASSWORD")
	}
	if stored != MysqlNativePassword(currentPwd) {
		return errors.New("WRONG_PASSWORD")
	}
	if len(newPwd) < minChangePwLength {
		return errors.New("비밀번호는 최소 8자 이상이어야 합니다")
	}
	if !pwChangeHasLetter.MatchString(newPwd) || !pwChangeHasNumber.MatchString(newPwd) || !pwChangeHasSpecial.MatchString(newPwd) {
		return errors.New("비밀번호는 영문, 숫자, 특수문자를 모두 포함해야 합니다")
	}
	return s.repo.UpdatePassword(usrSeq, MysqlNativePassword(newPwd))
}
