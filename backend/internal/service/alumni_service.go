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
		phone := ""
		if record.FMPhone.Valid {
			phone = record.FMPhone.String
		}
		email := ""
		if record.FMEmail.Valid {
			email = record.FMEmail.String
		}
		if record.FMSMS.Valid && strings.EqualFold(record.FMSMS.String, "N") {
			phone = maskPhone(phone)
		}
		if record.FMSpam.Valid && strings.EqualFold(record.FMSpam.String, "N") {
			email = maskEmail(email)
		}

		bizName := nullString(record.USRBizName)
		company := nullString(record.FMCompany)
		position := nullString(record.FMPos)

		// Fallback: if bizName is empty, use company name
		displayBizName := bizName
		if displayBizName == "" {
			displayBizName = company
		}

		// Fallback: if bizDesc is empty and position exists, show position
		bizDesc := nullString(record.USRBizDesc)
		if bizDesc == "" && position != "" {
			bizDesc = position
		}

		// Parse tags from GROUP_CONCAT result
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

		items = append(items, model.AlumniCard{
			FMSeq:       record.FMSEQ,
			FMName:      record.FMName,
			FMFN:        nullString(record.FMFN),
			FMDept:      nullString(record.FMDept),
			Company:     company,
			Position:    position,
			Phone:       phone,
			Email:       email,
			BizName:     displayBizName,
			BizDesc:     bizDesc,
			BizAddr:     nullString(record.USRBizAddr),
			JobCatName:  nullString(record.AJCName),
			JobCatColor: nullString(record.AJCColor),
			Tags:        tags,
			Photo:       nullString(record.USRPhoto),
			UsrSeq:      usrSeq,
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

func maskPhone(value string) string {
	if value == "" {
		return ""
	}
	if strings.Contains(value, "-") {
		parts := strings.Split(value, "-")
		if len(parts) >= 3 {
			parts[1] = "****"
			return strings.Join(parts, "-")
		}
	}
	runes := []rune(value)
	if len(runes) <= 4 {
		return strings.Repeat("*", len(runes))
	}
	prefix := string(runes[:2])
	suffix := string(runes[len(runes)-2:])
	return prefix + strings.Repeat("*", len(runes)-4) + suffix
}

func maskEmail(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if local == "" {
		return "***@" + parts[1]
	}
	masked := local[:1] + strings.Repeat("*", maxInt(1, len(local)-1))
	return masked + "@" + parts[1]
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
