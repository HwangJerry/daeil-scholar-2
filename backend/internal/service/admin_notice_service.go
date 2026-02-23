// Admin notice service — business logic for notice CRUD
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type AdminNoticeService struct {
	repo *repository.AdminNoticeRepository
}

func NewAdminNoticeService(repo *repository.AdminNoticeRepository) *AdminNoticeService {
	return &AdminNoticeService{repo: repo}
}

func (s *AdminNoticeService) List(page, size int, keyword string) ([]model.AdminNoticeRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repo.GetNotices(page, size, keyword)
}

func (s *AdminNoticeService) GetForEdit(seq int) (*model.NoticeDetail, error) {
	return s.repo.GetNoticeForEdit(seq)
}

func (s *AdminNoticeService) Create(subject, markdownText, regName string, usrSeq int, isPinned string) (int, error) {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return 0, err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	return s.repo.InsertNotice(&model.AdminNoticeInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
		IsPinned:     isPinned,
		RegName:      regName,
		USRSeq:       usrSeq,
	})
}

func (s *AdminNoticeService) Update(seq int, subject, markdownText, isPinned string) error {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	return s.repo.UpdateNotice(seq, &model.AdminNoticeInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
		IsPinned:     isPinned,
	})
}

func (s *AdminNoticeService) Delete(seq int) error {
	return s.repo.DeleteNotice(seq)
}

func (s *AdminNoticeService) TogglePin(seq int) error {
	return s.repo.TogglePin(seq)
}

func (s *AdminNoticeService) CountNotices() (int, error) {
	return s.repo.CountNotices()
}
