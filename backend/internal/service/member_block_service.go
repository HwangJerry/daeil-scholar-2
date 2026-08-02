package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var ErrMemberBlockTargetNotFound = errors.New("member block target is not an approved alumnus")

type MemberBlockQuerier interface {
	IsApprovedAlumni(userSeq int) (bool, error)
	List(blockerSeq int) ([]model.MemberBlockState, error)
	Get(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
	Block(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
	Unblock(blockerSeq, blockedSeq int) (*model.MemberBlockState, error)
}

// MemberBlockService manages the authenticated user's outbound block state.
type MemberBlockService struct {
	repo MemberBlockQuerier
}

func NewMemberBlockService(repo *repository.MemberBlockRepository) *MemberBlockService {
	return &MemberBlockService{repo: repo}
}

func (s *MemberBlockService) List(blockerSeq int) (*model.MemberBlockListResponse, error) {
	items, err := s.repo.List(blockerSeq)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.MemberBlockState{}
	}
	return &model.MemberBlockListResponse{Items: items}, nil
}

func (s *MemberBlockService) Get(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	if err := validateBlockParticipants(blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	return s.repo.Get(blockerSeq, blockedSeq)
}

func (s *MemberBlockService) Block(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	if err := validateBlockParticipants(blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	approved, err := s.repo.IsApprovedAlumni(blockedSeq)
	if err != nil {
		return nil, err
	}
	if !approved {
		return nil, ErrMemberBlockTargetNotFound
	}
	state, err := s.repo.Block(blockerSeq, blockedSeq)
	if errors.Is(err, repository.ErrMemberBlockTargetNotApproved) {
		return nil, ErrMemberBlockTargetNotFound
	}
	return state, err
}

func (s *MemberBlockService) Unblock(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	if err := validateBlockParticipants(blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	return s.repo.Unblock(blockerSeq, blockedSeq)
}

func validateBlockParticipants(blockerSeq, blockedSeq int) error {
	if blockerSeq <= 0 || blockedSeq <= 0 || blockerSeq == blockedSeq {
		return &model.ValidationError{Msg: "회원 식별자가 올바르지 않습니다"}
	}
	return nil
}
