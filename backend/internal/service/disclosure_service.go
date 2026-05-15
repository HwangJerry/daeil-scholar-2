// Disclosure service — public read-only orchestration for public-disclosure list/detail
package service

import (
	"strconv"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type DisclosureService struct {
	repo *repository.DisclosureRepository
}

func NewDisclosureService(repo *repository.DisclosureRepository) *DisclosureService {
	return &DisclosureService{repo: repo}
}

// List returns a page of disclosures plus hasMore/nextCursor metadata.
func (s *DisclosureService) List(cursor, size int) (*model.DisclosureListResponse, error) {
	if size <= 0 {
		size = 10
	}
	if size > 30 {
		size = 30
	}
	rows, err := s.repo.ListDisclosures(cursor, size)
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	resp := &model.DisclosureListResponse{Items: rows, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		resp.NextCursor = "seq_" + strconv.Itoa(last.SEQ)
	}
	return resp, nil
}

// GetDetail returns a disclosure detail with files; increments hit counter.
// Returns (nil, nil) if not found.
func (s *DisclosureService) GetDetail(seq int) (*model.NoticeDetail, error) {
	detail, err := s.repo.GetDisclosureDetail(seq)
	if err != nil || detail == nil {
		return detail, err
	}
	if err := s.repo.IncrementHit(seq); err != nil {
		return nil, err
	}
	files, err := s.repo.GetFilesByDisclosure(seq)
	if err != nil {
		return nil, err
	}
	detail.Files = files
	return detail, nil
}
