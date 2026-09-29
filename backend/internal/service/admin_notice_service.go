// Admin notice service — business logic for notice CRUD with attachment reconciliation
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// NoticePublishedNotifier is told about a newly published notice so it can
// broadcast a push. It is optional: push delivery is disabled by default, and a
// nil notifier makes publishing a notice a no-op for notifications.
type NoticePublishedNotifier interface {
	NotifyNoticePublished(noticeSeq int, subject string)
}

type AdminNoticeService struct {
	repo       *repository.AdminNoticeRepository
	fileRepo   *repository.FileRepository
	categories repository.AdminFeedCategoryStore
	notifier   NoticePublishedNotifier
}

// SetNoticePublishedNotifier wires the push broadcast after construction, the
// same way review notifications are wired, because the notifier only exists
// when push delivery is enabled.
func (s *AdminNoticeService) SetNoticePublishedNotifier(notifier NoticePublishedNotifier) {
	s.notifier = notifier
}

func NewAdminNoticeService(repo *repository.AdminNoticeRepository, fileRepo *repository.FileRepository, categories repository.AdminFeedCategoryStore) *AdminNoticeService {
	return &AdminNoticeService{repo: repo, fileRepo: fileRepo, categories: categories}
}

// List pages NOTICE posts; categorySeq > 0 filters to one category.
func (s *AdminNoticeService) List(page, size int, keyword string, categorySeq int) ([]model.AdminNoticeRow, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repo.GetNotices(page, size, keyword, categorySeq)
}

// checkCategory refuses a category seq that does not exist. Hidden categories
// are accepted: a post may keep (or be saved with) its hidden category.
func (s *AdminNoticeService) checkCategory(categorySeq *int) error {
	if categorySeq == nil {
		return nil
	}
	cat, err := s.categories.GetBySeq(*categorySeq)
	if err != nil {
		return err
	}
	if cat == nil {
		return ErrFeedCategoryUnknown
	}
	return nil
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

// Create publishes a Markdown notice. A nil categorySeq files it under the default category.
func (s *AdminNoticeService) Create(subject, markdownText, regName string, usrSeq int, isPinned string, attachedFileSeqs []int, categorySeq *int) (int, error) {
	if err := s.checkCategory(categorySeq); err != nil {
		return 0, err
	}
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return 0, err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	seq, err := s.repo.InsertNotice(&model.AdminNoticeInsert{
		Subject:         subject,
		Contents:        encoded,
		ContentsMD:      markdownText,
		Summary:         summary,
		ThumbnailURL:    thumbnail,
		IsPinned:        isPinned,
		RegName:         regName,
		USRSeq:          usrSeq,
		FeedCategorySeq: categorySeq,
	})
	if err != nil {
		return 0, err
	}
	attachErr := s.fileRepo.AttachFilesToNotice(seq, attachedFileSeqs)
	// The notice is live and readable the moment the insert commits, so it is
	// announced even when linking its attachments failed — the caller still
	// gets that error. Only a newly published notice is announced: editing or
	// pinning one is not news, so Update and TogglePin stay silent.
	if s.notifier != nil {
		s.notifier.NotifyNoticePublished(seq, subject)
	}
	return seq, attachErr
}

// Update rewrites a notice as Markdown. A nil categorySeq keeps its current category.
func (s *AdminNoticeService) Update(seq int, subject, markdownText, isPinned string, attachedFileSeqs []int, categorySeq *int) error {
	if err := s.checkCategory(categorySeq); err != nil {
		return err
	}
	encoded, summary, thumbnail, err := ConvertAndEncode(markdownText)
	if err != nil {
		return err
	}
	if isPinned == "" {
		isPinned = "N"
	}
	if err := s.repo.UpdateNotice(seq, &model.AdminNoticeInsert{
		Subject:         subject,
		Contents:        encoded,
		ContentsMD:      markdownText,
		Summary:         summary,
		ThumbnailURL:    thumbnail,
		IsPinned:        isPinned,
		FeedCategorySeq: categorySeq,
	}); err != nil {
		return err
	}
	return s.fileRepo.ReconcileAttachments(seq, attachedFileSeqs)
}

// SetCategory reclassifies a post without touching its content; legacy (HTML)
// posts, which the Markdown editor cannot save, use this.
func (s *AdminNoticeService) SetCategory(seq, categorySeq int) error {
	if err := s.checkCategory(&categorySeq); err != nil {
		return err
	}
	return s.repo.UpdateNoticeCategory(seq, categorySeq)
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
