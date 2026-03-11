// like_service.go — Business logic for like toggle operations
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// LikeService handles like-related business logic.
type LikeService struct {
	likeRepo repository.LikeQuerier
	feedRepo repository.FeedQuerier
	notifier *LikeNotifier
}

// NewLikeService creates a new LikeService.
func NewLikeService(likeRepo repository.LikeQuerier, feedRepo repository.FeedQuerier, notifSvc *NotificationService) *LikeService {
	return &LikeService{
		likeRepo: likeRepo,
		feedRepo: feedRepo,
		notifier: NewLikeNotifier(feedRepo, notifSvc),
	}
}

// ToggleLike toggles a like for a post.
// Uses HasUserLiked (COUNT-based) as source of truth and SetLikeOpenByUser
// for bulk updates to handle duplicate rows in WEO_BOARDLIKE.
func (s *LikeService) ToggleLike(bbsSeq int, usrSeq int, usrName string) (*model.LikeToggleResponse, error) {
	alreadyLiked, err := s.likeRepo.HasUserLiked(bbsSeq, usrSeq)
	if err != nil {
		return nil, err
	}

	var liked bool
	if alreadyLiked {
		// Unlike: set ALL rows for this user+post to 'N'
		if err := s.likeRepo.SetLikeOpenByUser(bbsSeq, usrSeq, "N"); err != nil {
			return nil, err
		}
		liked = false
	} else {
		// Like: check if any row exists (active or inactive)
		exists, findErr := s.likeRepo.HasAnyLikeRow(bbsSeq, usrSeq)
		if findErr != nil {
			return nil, findErr
		}

		if exists {
			// Reactivate: set ALL rows for this user+post to 'Y'
			if err := s.likeRepo.SetLikeOpenByUser(bbsSeq, usrSeq, "Y"); err != nil {
				return nil, err
			}
		} else {
			// First time: insert a new row
			if err := s.likeRepo.InsertLike(bbsSeq, usrSeq); err != nil {
				return nil, err
			}
		}
		liked = true
	}

	likeCnt, err := s.feedRepo.GetLikeCount(bbsSeq)
	if err != nil {
		return nil, err
	}

	if liked && s.notifier != nil {
		s.notifier.OnLiked(bbsSeq, usrSeq, usrName)
	}

	return &model.LikeToggleResponse{Liked: liked, LikeCnt: likeCnt}, nil
}

// HasUserLiked checks whether a user has liked a specific post.
func (s *LikeService) HasUserLiked(bbsSeq int, usrSeq int) (bool, error) {
	return s.likeRepo.HasUserLiked(bbsSeq, usrSeq)
}
