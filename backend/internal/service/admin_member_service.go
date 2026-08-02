// admin_member_service.go — Business logic for admin member management
package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var (
	ErrRejectionReasonRequired            = errors.New("rejection reason required")
	ErrLegacyVerificationStatusNotAllowed = errors.New("legacy verification status not allowed")
	ErrVerificationStale                  = repository.ErrVerificationStale
	ErrVerificationStateConflict          = repository.ErrVerificationStateConflict
)

var legacyVerificationStatuses = map[string]bool{
	"BAA": true,
	"BBB": true,
	"CCC": true,
	"ZZZ": true,
}

var allowedMemberStatuses = map[string]bool{
	"AAA": true, // 탈퇴회원
	"ABA": true, // 휴면회원
	"ACA": true, // 정지회원
	"BAA": true, // 승인거절
	"BBB": true, // 승인대기
	"CCC": true, // 승인회원
	"ZZZ": true, // 운영자
}

// AdminMemberService handles member management operations for administrators.
type AdminMemberService struct {
	repo *repository.AdminMemberRepository
}

// NewAdminMemberService creates an AdminMemberService.
func NewAdminMemberService(repo *repository.AdminMemberRepository) *AdminMemberService {
	return &AdminMemberService{repo: repo}
}

// List returns paginated member rows.
func (s *AdminMemberService) List(page, size int, q, fn, status string) ([]model.AdminMemberRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repo.GetMembers(page, size, q, fn, status)
}

// GetDetail returns the full detail for a single member.
func (s *AdminMemberService) GetDetail(seq int) (*model.AdminMemberDetail, error) {
	return s.repo.GetMemberDetail(seq)
}

func (s *AdminMemberService) ListAlumniVerifications(status string) ([]model.AdminAlumniVerification, error) {
	return s.repo.ListAlumniVerifications(strings.TrimSpace(status))
}

func (s *AdminMemberService) GetAlumniVerificationDetail(usrSeq int) (*model.AdminAlumniVerification, error) {
	return s.repo.GetAlumniVerificationDetail(usrSeq)
}

// UpdateStatus validates and applies a member status change.
func (s *AdminMemberService) UpdateStatus(seq int, status string) error {
	if legacyVerificationStatuses[status] {
		return ErrLegacyVerificationStatusNotAllowed
	}
	if !allowedMemberStatuses[status] {
		return fmt.Errorf("invalid member status: %s", status)
	}

	return s.repo.UpdateMemberStatus(seq, status)
}

func (s *AdminMemberService) RejectAlumniVerification(usrSeq int, reviewerSeq int, reason string, expectedUpdatedAt time.Time) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrRejectionReasonRequired
	}
	return s.repo.RejectAlumniVerification(usrSeq, reviewerSeq, reason, expectedUpdatedAt)
}

func (s *AdminMemberService) ApproveAlumniVerification(usrSeq int, reviewerSeq int, expectedUpdatedAt time.Time) error {
	return s.repo.ApproveAlumniVerification(usrSeq, reviewerSeq, expectedUpdatedAt)
}

// HasKakaoLink checks whether a member has a linked Kakao social account.
func (s *AdminMemberService) HasKakaoLink(usrSeq int) (bool, error) {
	return s.repo.HasKakaoLink(usrSeq)
}

// MemberStats holds aggregated member statistics.
type MemberStats struct {
	TotalMembers       int `json:"totalMembers"`
	KakaoLinkedMembers int `json:"kakaoLinkedMembers"`
	RecentLoginCount   int `json:"recentLoginCount"`
	PendingApprovals   int `json:"pendingApprovals"`
}

const recentLoginDays = 7

// GetMemberStats returns aggregated member statistics for the admin dashboard.
func (s *AdminMemberService) GetMemberStats() (*MemberStats, error) {
	total, err := s.repo.CountTotalMembers()
	if err != nil {
		return nil, err
	}
	kakao, err := s.repo.CountKakaoLinked()
	if err != nil {
		return nil, err
	}
	recent, err := s.repo.CountRecentLogins(recentLoginDays)
	if err != nil {
		return nil, err
	}
	pending, err := s.repo.CountPendingMembers()
	if err != nil {
		return nil, err
	}
	return &MemberStats{
		TotalMembers:       total,
		KakaoLinkedMembers: kakao,
		RecentLoginCount:   recent,
		PendingApprovals:   pending,
	}, nil
}
