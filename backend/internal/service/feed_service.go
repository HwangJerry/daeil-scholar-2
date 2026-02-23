// feed_service.go — Service layer for news feed: assembly, interleaving, hero notice, and ad retrieval with caching
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
	feedAdsCacheTTL  = 5 * time.Minute
	feedHeroCacheKey = "feed:hero"
	feedAdsCacheKey  = "feed:ads"
)

type FeedService struct {
	repo   *repository.FeedRepository
	adRepo *repository.AdRepository
	cache  *cache.Cache
}

func NewFeedService(repo *repository.FeedRepository, adRepo *repository.AdRepository, cacheStore *cache.Cache) *FeedService {
	return &FeedService{repo: repo, adRepo: adRepo, cache: cacheStore}
}

func (s *FeedService) GetFeed(cursor int, size int, excludeAdIDs []int, excludeSeq int, userSeq int) (*model.FeedResponse, error) {
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
	ads, err := s.getActiveAds(excludeAdIDs)
	if err != nil {
		ads = []model.AdItem{} // Non-fatal: render feed without ads
	}
	items := interleaveFeed(notices, ads)
	response := &model.FeedResponse{Items: items, HasMore: hasMore}
	if len(notices) > 0 {
		last := notices[len(notices)-1]
		response.NextCursor = "seq_" + strconv.Itoa(last.SEQ)
	}
	return response, nil
}

// getActiveAds returns the cached full ad list, filtering excludeIDs in-memory.
func (s *FeedService) getActiveAds(excludeIDs []int) ([]model.AdItem, error) {
	if v, ok := s.cache.Get(feedAdsCacheKey); ok {
		return filterAds(v.([]model.AdItem), excludeIDs), nil
	}
	all, err := s.adRepo.GetActiveAds(nil)
	if err != nil {
		return nil, err
	}
	s.cache.Set(feedAdsCacheKey, all, feedAdsCacheTTL)
	return filterAds(all, excludeIDs), nil
}

// filterAds returns ads excluding those whose MASeq is in excludeIDs.
func filterAds(ads []model.AdItem, excludeIDs []int) []model.AdItem {
	if len(excludeIDs) == 0 {
		return ads
	}
	excludeSet := make(map[int]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = struct{}{}
	}
	result := make([]model.AdItem, 0, len(ads))
	for _, ad := range ads {
		if _, skip := excludeSet[ad.MASeq]; !skip {
			result = append(result, ad)
		}
	}
	return result
}

func (s *FeedService) GetHero() (*model.NoticeItem, error) {
	if v, ok := s.cache.Get(feedHeroCacheKey); ok {
		return v.(*model.NoticeItem), nil
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
