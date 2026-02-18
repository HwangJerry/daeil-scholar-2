// Admin member service — business logic for member management
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var allowedMemberStatuses = map[string]bool{
	"AAA": true, // 탈퇴회원
	"ABA": true, // 휴면회원
	"ACA": true, // 정지회원
	"BAA": true, // 승인거절
	"BBB": true, // 승인대기
	"CCC": true, // 승인회원
	"ZZZ": true, // 운영자
}

type AdminMemberService struct {
	repo *repository.AdminMemberRepository
}

func NewAdminMemberService(repo *repository.AdminMemberRepository) *AdminMemberService {
	return &AdminMemberService{repo: repo}
}

func (s *AdminMemberService) List(page, size int, name, fn, status string) ([]model.AdminMemberRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repo.GetMembers(page, size, name, fn, status)
}

func (s *AdminMemberService) GetDetail(seq int) (*model.AdminMemberDetail, error) {
	return s.repo.GetMemberDetail(seq)
}

func (s *AdminMemberService) UpdateStatus(seq int, status string) error {
	if !allowedMemberStatuses[status] {
		return fmt.Errorf("invalid member status: %s", status)
	}
	return s.repo.UpdateMemberStatus(seq, status)
}

func (s *AdminMemberService) HasKakaoLink(usrSeq int) (bool, error) {
	return s.repo.HasKakaoLink(usrSeq)
}

// MemberStats holds aggregated member statistics.
type MemberStats struct {
	TotalMembers       int `json:"totalMembers"`
	KakaoLinkedMembers int `json:"kakaoLinkedMembers"`
	RecentLoginCount   int `json:"recentLoginCount"`
}

const recentLoginDays = 7

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
	return &MemberStats{
		TotalMembers:       total,
		KakaoLinkedMembers: kakao,
		RecentLoginCount:   recent,
	}, nil
}
