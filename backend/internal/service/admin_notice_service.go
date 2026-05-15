// Admin notice service — business logic for notice CRUD with attachment reconciliation
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type AdminNoticeService struct {
	repo     *repository.AdminNoticeRepository
	fileRepo *repository.FileRepository
}

func NewAdminNoticeService(repo *repository.AdminNoticeRepository, fileRepo *repository.FileRepository) *AdminNoticeService {
	return &AdminNoticeService{repo: repo, fileRepo: fileRepo}
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
	detail, err := s.repo.GetNoticeForEdit(seq)
	if err != nil {
		return nil, err
	}
	files, err := s.fileRepo.GetAttachmentsByNotice(seq)
	if err != nil {
		return nil, err
	}
	detail.Files = files
	return detail, nil
}

func (s *AdminNoticeService) Create(subject, markdownText, regName string, usrSeq int, isPinned string, attachedFileSeqs []int) (int, error) {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return 0, err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	seq, err := s.repo.InsertNotice(&model.AdminNoticeInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
		IsPinned:     isPinned,
		RegName:      regName,
		USRSeq:       usrSeq,
	})
	if err != nil {
		return 0, err
	}
	if err := s.fileRepo.AttachFilesToNotice(seq, attachedFileSeqs); err != nil {
		return seq, err
	}
	return seq, nil
}

func (s *AdminNoticeService) Update(seq int, subject, markdownText, isPinned string, attachedFileSeqs []int) error {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	if err := s.repo.UpdateNotice(seq, &model.AdminNoticeInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
		IsPinned:     isPinned,
	}); err != nil {
		return err
	}
	return s.fileRepo.ReconcileAttachments(seq, attachedFileSeqs)
}

func (s *AdminNoticeService) Delete(seq int) error {
	if err := s.repo.DeleteNotice(seq); err != nil {
		return err
	}
	return s.fileRepo.SoftDeleteFilesByJoin(seq)
}

func (s *AdminNoticeService) TogglePin(seq int) error {
	return s.repo.TogglePin(seq)
}

func (s *AdminNoticeService) CountNotices() (int, error) {
	return s.repo.CountNotices()
}
