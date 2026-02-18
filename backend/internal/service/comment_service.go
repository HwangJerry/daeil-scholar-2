// comment_service.go — Business logic for comment operations
package service

import (
	"errors"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

const maxCommentLength = 500

// CommentService handles comment-related business logic.
type CommentService struct {
	commentRepo *repository.CommentRepository
}

// NewCommentService creates a new CommentService.
func NewCommentService(commentRepo *repository.CommentRepository) *CommentService {
	return &CommentService{commentRepo: commentRepo}
}

// GetComments returns all visible comments for a post.
func (s *CommentService) GetComments(joinSeq int) ([]model.Comment, error) {
	return s.commentRepo.GetComments(joinSeq)
}

// AddComment validates and inserts a new comment, returning it.
func (s *CommentService) AddComment(joinSeq int, usrSeq int, regName string, contents string) (*model.Comment, error) {
	if contents == "" {
		return nil, errors.New("댓글 내용을 입력해주세요")
	}
	if utf8.RuneCountInString(contents) > maxCommentLength {
		return nil, errors.New("댓글은 500자 이내로 작성해주세요")
	}

	lastID, err := s.commentRepo.InsertComment(joinSeq, usrSeq, regName, contents)
	if err != nil {
		return nil, err
	}

	return &model.Comment{
		BCSeq:    int(lastID),
		JoinSeq:  joinSeq,
		USRSeq:   usrSeq,
		RegName:  regName,
		Contents: contents,
	}, nil
}

// DeleteComment soft-deletes a comment owned by the user.
func (s *CommentService) DeleteComment(bcSeq int, usrSeq int) error {
	affected, err := s.commentRepo.SoftDeleteComment(bcSeq, usrSeq)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("댓글을 삭제할 수 없습니다")
	}
	return nil
}
