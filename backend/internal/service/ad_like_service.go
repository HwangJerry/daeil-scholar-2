// ad_like_service.go — Business logic for ad like toggle operations
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// AdLikeService handles ad like toggle business logic.
type AdLikeService struct {
	likeRepo *repository.AdLikeRepository
}

// NewAdLikeService creates a new AdLikeService.
func NewAdLikeService(likeRepo *repository.AdLikeRepository) *AdLikeService {
	return &AdLikeService{likeRepo: likeRepo}
}

// ToggleLike toggles a like for an ad and returns the new liked state and count.
// Mirrors the pattern from LikeService.ToggleLike.
func (s *AdLikeService) ToggleLike(maSeq, usrSeq int) (*model.LikeToggleResponse, error) {
	alreadyLiked, err := s.likeRepo.HasUserLikedAd(maSeq, usrSeq)
	if err != nil {
		return nil, err
	}

	var liked bool
	if alreadyLiked {
		if err := s.likeRepo.SetAdLikeOpenByUser(maSeq, usrSeq, "N"); err != nil {
			return nil, err
		}
		liked = false
	} else {
		exists, findErr := s.likeRepo.HasAnyAdLikeRow(maSeq, usrSeq)
		if findErr != nil {
			return nil, findErr
		}
		if exists {
			if err := s.likeRepo.SetAdLikeOpenByUser(maSeq, usrSeq, "Y"); err != nil {
				return nil, err
			}
		} else {
			if err := s.likeRepo.InsertAdLike(maSeq, usrSeq); err != nil {
				return nil, err
			}
		}
		liked = true
	}

	likeCnt, err := s.likeRepo.GetAdLikeCount(maSeq)
	if err != nil {
		return nil, err
	}

	return &model.LikeToggleResponse{Liked: liked, LikeCnt: likeCnt}, nil
}
