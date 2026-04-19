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
	weeklyCount, err := s.repo.GetWeeklyCount()
	if err != nil {
		weeklyCount = 0
	}
	items := make([]model.AlumniCard, 0, len(records))
	for _, record := range records {
		var tags []string
		if record.Tags.Valid && record.Tags.String != "" {
			tags = strings.Split(record.Tags.String, ",")
		}
		if tags == nil {
			tags = []string{}
		}

		usrSeq := 0
		if record.USRSeq.Valid {
			usrSeq = int(record.USRSeq.Int64)
		}

		phone := nullString(record.USRPhone)
		if record.USRPhonePublic.Valid && record.USRPhonePublic.String == "N" {
			phone = ""
		}
		email := nullString(record.USREmail)
		if record.USREmailPublic.Valid && record.USREmailPublic.String == "N" {
			email = ""
		}

		items = append(items, model.AlumniCard{
			FMSeq:       usrSeq,
			FMName:      record.USRName,
			FMFN:        nullString(record.USRFN),
			FMDept:      nullString(record.USRDept),
			BizName:     nullString(record.USRBizName),
			BizDesc:     nullString(record.USRBizDesc),
			BizAddr:     nullString(record.USRBizAddr),
			Position:    nullString(record.USRPosition),
			Phone:       phone,
			Email:       email,
			JobCatName:  nullString(record.AJCName),
			Tags:        tags,
			Photo:       nullString(record.USRPhoto),
			UsrSeq:      usrSeq,
			Nick:        nullString(record.USRNick),
			BizCard:     nullString(record.USRBizCard),
		})
	}
	totalPages := 0
	if params.Size > 0 {
		totalPages = (total + params.Size - 1) / params.Size
	}
	return &model.AlumniSearchResponse{
		Items:       items,
		TotalCount:  total,
		WeeklyCount: weeklyCount,
		Page:        params.Page,
		Size:        params.Size,
		TotalPages:  totalPages,
	}, nil
}

const widgetPreviewCacheKey = "alumni:widget:preview"

// GetWidgetPreview returns a cached minimal alumni list + total count for the public widget.
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

