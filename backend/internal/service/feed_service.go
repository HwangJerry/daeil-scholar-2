// feed_service.go — Service layer for news feed assembly and hero notice caching
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

const (
	feedHeroCacheTTL = 2 * time.Minute
	feedHeroCacheKey = "feed:hero"
)

type FeedService struct {
	repo  repository.FeedQuerier
	cache *cache.Cache
}

func NewFeedService(repo repository.FeedQuerier, cacheStore *cache.Cache) *FeedService {
	return &FeedService{repo: repo, cache: cacheStore}
}

func (s *FeedService) GetFeed(cursor int, size int, excludeSeq int, userSeq int) (*model.FeedResponse, error) {
	if size <= 0 {
		size = 10
	}
	if size > 20 {
		size = 20
	}
	notices, err := s.repo.GetNotices(cursor, size, excludeSeq, userSeq)
	if err != nil {
		return nil, err
	}
	hasMore := len(notices) > size
	if hasMore {
		notices = notices[:size]
	}
	items := make([]model.FeedItem, 0, len(notices))
	for i := range notices {
		items = append(items, model.FeedItem{
			Type:       "notice",
			NoticeItem: &notices[i],
		})
	}
	response := &model.FeedResponse{Items: items, HasMore: hasMore}
	if len(notices) > 0 {
		last := notices[len(notices)-1]
		response.NextCursor = "seq_" + strconv.Itoa(last.SEQ)
	}
	return response, nil
}

func (s *FeedService) GetHero() (*model.NoticeItem, error) {
	if v, ok := s.cache.Get(feedHeroCacheKey); ok {
		if hero, ok := v.(*model.NoticeItem); ok {
			return hero, nil
		}
		s.cache.Delete(feedHeroCacheKey)
	}
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
	s.cache.Set(feedHeroCacheKey, hero, feedHeroCacheTTL)
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
