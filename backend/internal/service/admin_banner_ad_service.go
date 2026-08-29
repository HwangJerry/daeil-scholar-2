package service

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

var (
	ErrActiveConflict      = repository.ErrActiveConflict
	ErrInvalidBannerURL    = errors.New("invalid_banner_url")
	ErrInvalidOpenYN       = errors.New("invalid_open_yn")
	ErrInvalidBannerPeriod = errors.New("invalid_banner_period")
	ErrActiveWithoutImages = errors.New("active_banner_without_images")
)

const bannerDateTimeLayout = "2006-01-02 15:04:05"

type AdminBannerAdService struct {
	repo        *repository.AdminBannerAdRepository
	fileStorage *FileStorageService
}

func NewAdminBannerAdService(repo *repository.AdminBannerAdRepository, fileStorage ...*FileStorageService) *AdminBannerAdService {
	service := &AdminBannerAdService{repo: repo}
	if len(fileStorage) > 0 {
		service.fileStorage = fileStorage[0]
	}
	return service
}

func (s *AdminBannerAdService) List() ([]model.AdminBannerAdRow, error) {
	return s.repo.GetBanners()
}

func (s *AdminBannerAdService) Get(seq int) (*model.AdminBannerAdRow, error) {
	return s.repo.GetBanner(seq)
}

func (s *AdminBannerAdService) Create(a *model.AdminBannerAdInsert) (int, error) {
	if err := validateBannerAd(a); err != nil {
		return 0, err
	}
	return s.repo.InsertBanner(a)
}

func (s *AdminBannerAdService) Update(seq int, a *model.AdminBannerAdInsert) error {
	if err := validateBannerAd(a); err != nil {
		return err
	}
	removedImageURLs, err := s.repo.UpdateBanner(seq, a)
	if err != nil {
		return err
	}
	return s.deleteUploadedImages(removedImageURLs)
}

func (s *AdminBannerAdService) Delete(seq int) error {
	removedImageURLs, err := s.repo.DeleteBanner(seq)
	if err != nil {
		return err
	}
	return s.deleteUploadedImages(removedImageURLs)
}

func (s *AdminBannerAdService) GetStats(bnSeq int) (*model.AdminBannerAdStats, error) {
	return s.repo.GetBannerStats(bnSeq)
}

func validateBannerAd(a *model.AdminBannerAdInsert) error {
	trimmedURL := strings.TrimSpace(a.BNURL)
	parsedURL, err := url.ParseRequestURI(trimmedURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return ErrInvalidBannerURL
	}
	a.BNURL = trimmedURL

	if a.OpenYN != "Y" && a.OpenYN != "N" {
		return ErrInvalidOpenYN
	}
	if a.BNStartDate != nil && a.BNEndDate != nil {
		startDate, startErr := time.Parse(bannerDateTimeLayout, *a.BNStartDate)
		endDate, endErr := time.Parse(bannerDateTimeLayout, *a.BNEndDate)
		if startErr != nil || endErr != nil || startDate.After(endDate) {
			return ErrInvalidBannerPeriod
		}
	}
	if a.OpenYN == "Y" {
		hasImage := false
		for _, imageURL := range a.ImageURLs {
			if strings.TrimSpace(imageURL) != "" {
				hasImage = true
				break
			}
		}
		if !hasImage {
			return ErrActiveWithoutImages
		}
	}
	return nil
}

func (s *AdminBannerAdService) deleteUploadedImages(imageURLs []string) error {
	if s.fileStorage == nil {
		return nil
	}

	deleteErrors := make([]error, 0)
	for _, imageURL := range imageURLs {
		if err := s.fileStorage.DeleteUploadedURL(imageURL, "bannerAd"); err != nil {
			deleteErrors = append(deleteErrors, err)
		}
	}
	return errors.Join(deleteErrors...)
}
