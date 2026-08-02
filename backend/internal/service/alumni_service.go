package service

import (
	"database/sql"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

type AlumniService struct {
	repo  *repository.AlumniRepository
	cache *cache.Cache
}

func NewAlumniService(repo *repository.AlumniRepository, cacheStore *cache.Cache) *AlumniService {
	return &AlumniService{repo: repo, cache: cacheStore}
}

func (s *AlumniService) Search(params model.AlumniSearchParams) (*model.AlumniSearchResponse, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Cohort = strings.TrimSpace(params.Cohort)
	params.Department = strings.TrimSpace(params.Department)
	params.JobRole = strings.TrimSpace(params.JobRole)
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	if params.Size > 50 {
		params.Size = 50
	}
	records, total, err := s.repo.Search(params)
	if err != nil {
		return nil, err
	}
	items := make([]model.AlumniCard, 0, len(records))
	for _, record := range records {
		items = append(items, model.AlumniCard{
			UserSeq:     record.USRSeq,
			Name:        record.USRName,
			PhotoURL:    nullableString(record.USRPhoto),
			Cohort:      nullString(record.Cohort),
			Department:  nullString(record.Department),
			JobCategory: nullString(record.AJCName),
			JobRole:     nullString(record.USRPosition),
		})
	}
	totalPages := 0
	if params.Size > 0 {
		totalPages = (total + params.Size - 1) / params.Size
	}
	return &model.AlumniSearchResponse{
		Items:      items,
		Page:       params.Page,
		Size:       params.Size,
		TotalCount: total,
		TotalPages: totalPages,
	}, nil
}

func (s *AlumniService) GetDetail(viewerSeq, userSeq int) (*model.AlumniDetail, error) {
	record, err := s.repo.GetDetail(viewerSeq, userSeq)
	if err != nil || record == nil {
		return nil, err
	}
	return &model.AlumniDetail{
		UserSeq:     record.USRSeq,
		Name:        record.USRName,
		PhotoURL:    nullableString(record.USRPhoto),
		Cohort:      nullString(record.Cohort),
		Department:  nullString(record.Department),
		JobCategory: nullString(record.AJCName),
		JobRole:     nullString(record.USRPosition),
		Phone:       publicValue(record.USRPhone, record.USRPhonePublic),
		Email:       publicValue(record.USREmail, record.USREmailPublic),
		BlockState: model.AlumniBlockState{
			BlockedByMe: record.BlockedByMe,
		},
	}, nil
}

const widgetPreviewCacheKey = "alumni:widget:preview"

// GetWidgetPreview returns a cached minimal approved-alumni list and total count.
func (s *AlumniService) GetWidgetPreview() (*model.AlumniWidgetResponse, error) {
	if cached, found := s.cache.Get(widgetPreviewCacheKey); found {
		return cached.(*model.AlumniWidgetResponse), nil
	}
	names, total, err := s.repo.GetWidgetPreview()
	if err != nil {
		return nil, err
	}
	items := make([]model.AlumniWidgetItem, 0, len(names))
	for _, name := range names {
		items = append(items, model.AlumniWidgetItem{FmName: name})
	}
	result := &model.AlumniWidgetResponse{Items: items, TotalCount: total}
	s.cache.Set(widgetPreviewCacheKey, result, 10*time.Minute)
	return result, nil
}

// GetJobCategories returns all active job categories for public consumption (e.g., registration form).
func (s *AlumniService) GetJobCategories() ([]model.JobCategory, error) {
	if cached, found := s.cache.Get("alumni_job_categories"); found {
		if cats, ok := cached.([]model.JobCategory); ok {
			return cats, nil
		}
	}
	cats, err := s.repo.GetJobCategories()
	if err != nil {
		return nil, err
	}
	s.cache.Set("alumni_job_categories", cats, time.Hour)
	return cats, nil
}

func (s *AlumniService) GetFilters() (*model.AlumniFilters, error) {
	if cached, found := s.cache.Get("alumni_filters"); found {
		if filters, ok := cached.(*model.AlumniFilters); ok {
			return filters, nil
		}
	}
	filters, err := s.repo.GetFilters()
	if err != nil {
		return nil, err
	}
	s.cache.Set("alumni_filters", filters, time.Hour)
	return filters, nil
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := value.String
	return &result
}

func publicValue(value, public sql.NullString) *string {
	if !public.Valid || public.String != "Y" {
		return nil
	}
	return nullableString(value)
}
