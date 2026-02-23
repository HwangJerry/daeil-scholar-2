// ad_comment_service.go — Business logic for ad comment CRUD operations
package service

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// AdCommentService handles ad comment business logic.
type AdCommentService struct {
	commentRepo *repository.AdCommentRepository
}

// NewAdCommentService creates a new AdCommentService.
func NewAdCommentService(commentRepo *repository.AdCommentRepository) *AdCommentService {
	return &AdCommentService{commentRepo: commentRepo}
}

// ListComments returns all active comments for an ad.
func (s *AdCommentService) ListComments(maSeq int) ([]model.AdComment, error) {
	return s.commentRepo.ListAdComments(maSeq)
}

// CreateComment validates and inserts a new comment for an ad.
func (s *AdCommentService) CreateComment(maSeq, usrSeq int, nickname, contents string) (model.AdComment, error) {
	if contents == "" {
		return model.AdComment{}, errors.New("댓글 내용을 입력해주세요")
	}
	return s.commentRepo.InsertAdComment(maSeq, usrSeq, nickname, contents)
}

// DeleteComment soft-deletes a comment owned by usrSeq.
func (s *AdCommentService) DeleteComment(acSeq, usrSeq int) error {
	return s.commentRepo.DeleteAdComment(acSeq, usrSeq)
}
