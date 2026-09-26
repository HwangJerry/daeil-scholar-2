// admin_operator_service.go — Root-only rules for granting, changing and revoking admin roles
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidAdminRole         = errors.New("invalid admin role")
	ErrSelfAdminRoleChange      = errors.New("admins cannot change their own role")
	ErrOperatorTargetIneligible = repository.ErrOperatorTargetIneligible
	ErrOperatorNotFound         = repository.ErrOperatorNotFound
	ErrLastRootAdmin            = repository.ErrLastRootAdmin
)

type AdminOperatorService struct {
	repo *repository.AdminOperatorRepository
}

func NewAdminOperatorService(repo *repository.AdminOperatorRepository) *AdminOperatorService {
	return &AdminOperatorService{repo: repo}
}

func (s *AdminOperatorService) List() ([]model.AdminOperator, error) {
	return s.repo.ListOperators()
}

// SetRole grants or changes targetSeq's admin role. A root cannot change their
// own role, which also keeps a lone root from locking everyone out.
func (s *AdminOperatorService) SetRole(actorSeq, targetSeq int, role model.AdminRole) error {
	if !role.Valid() {
		return ErrInvalidAdminRole
	}
	if actorSeq == targetSeq {
		return ErrSelfAdminRoleChange
	}
	if err := s.repo.SetOperatorRole(actorSeq, targetSeq, role); err != nil {
		return err
	}
	log.Info().Int("actorSeq", actorSeq).Int("targetSeq", targetSeq).Str("role", string(role)).Msg("admin role set")
	return nil
}

// Revoke removes targetSeq's admin role.
func (s *AdminOperatorService) Revoke(actorSeq, targetSeq int) error {
	if actorSeq == targetSeq {
		return ErrSelfAdminRoleChange
	}
	if err := s.repo.RevokeOperator(targetSeq); err != nil {
		return err
	}
	log.Info().Int("actorSeq", actorSeq).Int("targetSeq", targetSeq).Msg("admin role revoked")
	return nil
}
