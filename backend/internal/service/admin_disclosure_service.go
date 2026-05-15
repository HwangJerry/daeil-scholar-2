// Admin disclosure service — business logic for public-disclosure CRUD with attachment reconciliation
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type AdminDisclosureService struct {
	repo     *repository.AdminDisclosureRepository
	fileRepo *repository.FileRepository
}

func NewAdminDisclosureService(repo *repository.AdminDisclosureRepository, fileRepo *repository.FileRepository) *AdminDisclosureService {
	return &AdminDisclosureService{repo: repo, fileRepo: fileRepo}
}

func (s *AdminDisclosureService) List(page, size int, keyword string) ([]model.AdminDisclosureRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repo.GetDisclosures(page, size, keyword)
}

func (s *AdminDisclosureService) GetForEdit(seq int) (*model.NoticeDetail, error) {
	detail, err := s.repo.GetDisclosureForEdit(seq)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	files, err := s.fileRepo.GetAttachmentsByNotice(seq)
	if err != nil {
		return nil, err
	}
	detail.Files = files
	return detail, nil
}

func (s *AdminDisclosureService) Create(subject, markdownText, regName string, usrSeq int, attachedFileSeqs []int) (int, error) {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return 0, err
	}
	seq, err := s.repo.InsertDisclosure(&model.AdminDisclosureInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
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

func (s *AdminDisclosureService) Update(seq int, subject, markdownText string, attachedFileSeqs []int) error {
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateDisclosure(seq, &model.AdminDisclosureInsert{
		Subject:      subject,
		Contents:     encoded,
		ContentsMD:   markdownText,
		Summary:      summary,
		ThumbnailURL: thumbnail,
	}); err != nil {
		return err
	}
	return s.fileRepo.ReconcileAttachments(seq, attachedFileSeqs)
}

func (s *AdminDisclosureService) Delete(seq int) error {
	if err := s.repo.DeleteDisclosure(seq); err != nil {
		return err
	}
	return s.fileRepo.SoftDeleteFilesByJoin(seq)
}

func (s *AdminDisclosureService) CountDisclosures() (int, error) {
	return s.repo.CountDisclosures()
}
