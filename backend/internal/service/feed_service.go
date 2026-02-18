package service

import (
	"strconv"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

type FeedService struct {
	repo   *repository.FeedRepository
	adRepo *repository.AdRepository
	cache  *cache.Cache
}

func NewFeedService(repo *repository.FeedRepository, adRepo *repository.AdRepository, cacheStore *cache.Cache) *FeedService {
	return &FeedService{repo: repo, adRepo: adRepo, cache: cacheStore}
}

func (s *FeedService) GetFeed(cursor int, size int, excludeAdIDs []int, excludeSeq int) (*model.FeedResponse, error) {
	if size <= 0 {
		size = 10
	}
	if size > 20 {
		size = 20
	}
	notices, err := s.repo.GetNotices(cursor, size, excludeSeq)
	if err != nil {
		return nil, err
	}
	hasMore := len(notices) > size
	if hasMore {
		notices = notices[:size]
	}
	ads, err := s.adRepo.GetActiveAds(excludeAdIDs)
	if err != nil {
		return nil, err
	}
	items := interleaveFeed(notices, ads)
	response := &model.FeedResponse{Items: items, HasMore: hasMore}
	if len(notices) > 0 {
		last := notices[len(notices)-1]
		response.NextCursor = "seq_" + strconv.Itoa(last.SEQ)
	}
	return response, nil
}

func (s *FeedService) GetHero() (*model.NoticeItem, error) {
	hero, err := s.repo.GetHeroNotice()
	if err != nil || hero == nil {
		return hero, err
	}
	// Enrichment is non-fatal: hero still renders without like/comment counts
	if likeCnt, err := s.repo.GetLikeCount(hero.SEQ); err == nil {
		hero.LikeCnt = likeCnt
	}
	if commentCnt, err := s.repo.GetCommentCount(hero.SEQ); err == nil {
		hero.CommentCnt = commentCnt
	}
	return hero, nil
}

func (s *FeedService) GetNoticeDetail(seq int) (*model.NoticeDetail, error) {
	detail, err := s.repo.GetNoticeDetail(seq)
	if err != nil || detail == nil {
		return detail, err
	}
	if err := s.repo.IncrementHit(seq); err != nil {
		return nil, err
	}
	likeCnt, err := s.repo.GetLikeCount(seq)
	if err != nil {
		return nil, err
	}
	commentCnt, err := s.repo.GetCommentCount(seq)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.GetFilesByPost(seq)
	if err != nil {
		return nil, err
	}
	detail.LikeCnt = likeCnt
	detail.CommentCnt = commentCnt
	detail.Files = files
	return detail, nil
}

func (s *FeedService) GetPostSiblings(seq int) (*model.PostSiblings, error) {
	prev, err := s.repo.GetPrevPost(seq)
	if err != nil {
		return nil, err
	}
	next, err := s.repo.GetNextPost(seq)
	if err != nil {
		return nil, err
	}
	return &model.PostSiblings{Prev: prev, Next: next}, nil
}

func interleaveFeed(notices []model.NoticeItem, ads []model.AdItem) []model.FeedItem {
	items := make([]model.FeedItem, 0, len(notices)+len(ads))
	adIndex := 0
	for i := range notices {
		notice := notices[i]
		items = append(items, model.FeedItem{Type: "notice", NoticeItem: &notice})
		if len(notices) >= 4 && (i+1)%4 == 0 && adIndex < len(ads) {
			ad := ads[adIndex]
			adIndex++
			items = append(items, model.FeedItem{Type: "ad", AdItem: &ad})
		}
	}
	return items
}
