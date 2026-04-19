// admin_job_category_service.go — Business logic for job category admin management
package service

import (
	"errors"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

// ErrDuplicateCategoryName is returned when the category name already exists.
var ErrDuplicateCategoryName = errors.New("duplicate_category_name")

type AdminJobCategoryService struct {
	repo  *repository.AdminJobCategoryRepository
	cache *cache.Cache
}

func NewAdminJobCategoryService(repo *repository.AdminJobCategoryRepository, cacheStore *cache.Cache) *AdminJobCategoryService {
	return &AdminJobCategoryService{repo: repo, cache: cacheStore}
}

func (s *AdminJobCategoryService) List() ([]model.AdminJobCategory, error) {
	return s.repo.GetAll()
}

func (s *AdminJobCategoryService) Create(req model.AdminJobCategoryUpsert) (int64, error) {
	id, err := s.repo.Insert(req)
	if err != nil {
		if isDuplicateEntry(err) {
			return 0, ErrDuplicateCategoryName
		}
		return 0, err
	}
	s.cache.Delete("alumni_job_categories")
	return id, nil
}

func (s *AdminJobCategoryService) Update(seq int, req model.AdminJobCategoryUpsert) error {
	if err := s.repo.Update(seq, req); err != nil {
		if isDuplicateEntry(err) {
			return ErrDuplicateCategoryName
		}
		return err
	}
	s.cache.Delete("alumni_job_categories")
	return nil
}

func (s *AdminJobCategoryService) Delete(seq int) error {
	if err := s.repo.SoftDelete(seq); err != nil {
		return err
	}
	s.cache.Delete("alumni_job_categories")
	return nil
}

// isDuplicateEntry detects MariaDB duplicate key errors (error 1062).
func isDuplicateEntry(err error) bool {
	return strings.Contains(err.Error(), "1062")
}
