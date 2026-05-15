// history_service.go — Business logic for history entry management
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type HistoryService struct {
	repo *repository.HistoryRepository
}

func NewHistoryService(repo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{repo: repo}
}

// GetGrouped returns all entries grouped by year, newest year first.
func (s *HistoryService) GetGrouped() ([]model.HistoryYearGroup, error) {
	entries, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	// Group into years while preserving order (entries are already DESC by date).
	var groups []model.HistoryYearGroup
	yearIndex := map[int]int{} // year → index in groups slice

	for _, e := range entries {
		year := yearFromDate(e.HEEventDate)
		idx, exists := yearIndex[year]
		if !exists {
			groups = append(groups, model.HistoryYearGroup{Year: year, Items: []model.HistoryEntry{}})
			idx = len(groups) - 1
			yearIndex[year] = idx
		}
		groups[idx].Items = append(groups[idx].Items, e)
	}
	return groups, nil
}

// GetAll returns all entries flat (for admin list).
func (s *HistoryService) GetAll() ([]model.HistoryEntry, error) {
	return s.repo.GetAll()
}

// Create validates and inserts a new entry, returning its seq.
func (s *HistoryService) Create(req model.HistoryUpsertRequest) (int64, error) {
	if err := validateHistoryUpsert(req); err != nil {
		return 0, err
	}
	return s.repo.Insert(req)
}

// Update validates and updates an existing entry.
func (s *HistoryService) Update(seq int, req model.HistoryUpsertRequest) error {
	if err := validateHistoryUpsert(req); err != nil {
		return err
	}
	return s.repo.Update(seq, req)
}

// Delete permanently removes an entry.
func (s *HistoryService) Delete(seq int) error {
	return s.repo.Delete(seq)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// yearFromDate extracts the year from a "YYYY-MM-DD" string without importing time.
func yearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	y := 0
	for _, c := range date[:4] {
		if c < '0' || c > '9' {
			return 0
		}
		y = y*10 + int(c-'0')
	}
	return y
}

func validateHistoryUpsert(req model.HistoryUpsertRequest) error {
	if req.EventDate == "" {
		return &model.ValidationError{Msg: "날짜를 입력해주세요"}
	}
	if req.Text == "" {
		return &model.ValidationError{Msg: "내용을 입력해주세요"}
	}
	if len([]rune(req.Text)) > 500 {
		return &model.ValidationError{Msg: "내용은 500자 이하로 입력해주세요"}
	}
	return nil
}
