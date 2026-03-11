// password_reset_service.go — Business logic for password reset flow
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
)

const (
	resetTokenBytes   = 32
	resetTokenExpiry  = 30 * time.Minute
	minPasswordLength = 4
)

// PasswordResetService handles the password reset request and confirmation flow.
type PasswordResetService struct {
	repo        repository.PasswordResetQuerier
	emailQueue  chan<- model.EmailMessage
	logger      zerolog.Logger
	siteBaseURL string
}

// NewPasswordResetService creates a PasswordResetService with the given dependencies.
func NewPasswordResetService(
	repo repository.PasswordResetQuerier,
	emailQueue chan<- model.EmailMessage,
	logger zerolog.Logger,
	siteBaseURL string,
) *PasswordResetService {
	return &PasswordResetService{
		repo:        repo,
		emailQueue:  emailQueue,
		logger:      logger,
		siteBaseURL: siteBaseURL,
	}
}

// RequestReset looks up a member by email, generates a reset token, and enqueues a reset email.
// Always returns nil to avoid revealing whether the email exists (security best practice).
func (s *PasswordResetService) RequestReset(req model.PasswordResetRequest) error {
	if req.Email == "" {
		return errors.New("이메일을 입력해주세요")
	}

	user, err := s.repo.FindMemberByEmail(req.Email)
	if err != nil {
		s.logger.Error().Err(err).Str("email", req.Email).Msg("password reset: email lookup failed")
		return nil
	}
	if user == nil {
		s.logger.Debug().Str("email", req.Email).Msg("password reset: no member found for email")
		return nil
	}

	tokenBytes := make([]byte, resetTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		s.logger.Error().Err(err).Msg("password reset: token generation failed")
		return nil
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(resetTokenExpiry)

	if err := s.repo.InsertToken(user.USRSeq, token, expiresAt); err != nil {
		s.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("password reset: token insert failed")
		return nil
	}

	resetURL := s.siteBaseURL + "/reset-password?token=" + token
	body, renderErr := RenderPasswordResetEmail(PasswordResetEmailData{
		Name:     user.USRName,
		ResetURL: resetURL,
	})
	if renderErr != nil {
		s.logger.Error().Err(renderErr).Msg("password reset: email render failed")
		return nil
	}

	msg := model.EmailMessage{
		To:      req.Email,
		Subject: "비밀번호 재설정",
		Body:    body,
	}
	select {
	case s.emailQueue <- msg:
		s.logger.Info().Str("email", req.Email).Msg("password reset email enqueued")
	default:
		s.logger.Warn().Str("email", req.Email).Msg("password reset: email queue full")
	}

	return nil
}

// ConfirmReset validates the token, hashes the new password, and updates the member's password.
func (s *PasswordResetService) ConfirmReset(req model.PasswordResetConfirm) error {
	if req.Token == "" {
		return errors.New("토큰이 없습니다")
	}
	if len(req.NewPassword) < minPasswordLength {
		return errors.New("비밀번호는 최소 4자 이상이어야 합니다")
	}

	tokenRow, err := s.repo.FindValidToken(req.Token)
	if err != nil {
		return errors.New("비밀번호 재설정 처리에 실패했습니다")
	}
	if tokenRow == nil {
		return errors.New("유효하지 않거나 만료된 링크입니다")
	}

	hashed := MysqlNativePassword(req.NewPassword)
	if err := s.repo.UpdatePassword(tokenRow.USRSeq, hashed); err != nil {
		s.logger.Error().Err(err).Int("usrSeq", tokenRow.USRSeq).Msg("password reset: password update failed")
		return errors.New("비밀번호 변경에 실패했습니다")
	}

	if err := s.repo.MarkTokenUsed(req.Token); err != nil {
		s.logger.Error().Err(err).Msg("password reset: mark token used failed")
	}

	return nil
}

// ValidateToken checks whether a reset token is still valid and returns the member name.
func (s *PasswordResetService) ValidateToken(token string) (*model.ValidateTokenResponse, error) {
	if token == "" {
		return &model.ValidateTokenResponse{Valid: false}, nil
	}

	tokenRow, err := s.repo.FindValidToken(token)
	if err != nil {
		return nil, err
	}
	if tokenRow == nil {
		return &model.ValidateTokenResponse{Valid: false}, nil
	}

	name, err := s.repo.GetMemberNameBySeq(tokenRow.USRSeq)
	if err != nil {
		return &model.ValidateTokenResponse{Valid: true}, nil
	}

	return &model.ValidateTokenResponse{Valid: true, Name: name}, nil
}
