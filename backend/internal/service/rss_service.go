// rss_service.go — Service for generating RSS feed entries from public posts
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

const (
	rssCacheKey = "rss:posts"
	rssCacheTTL = 1 * time.Hour
	rssMaxItems = 100
)

// RSSEntry is the view type for the RSS handler.
type RSSEntry struct {
	SeqStr   string
	PubDate  time.Time
	Subject  string
	Summary  string
}

// RSSService provides cached access to public post data for RSS 2.0 generation.
type RSSService struct {
	repo  *repository.FeedRepository
	cache *cache.Cache
}

// NewRSSService creates a new RSSService.
func NewRSSService(repo *repository.FeedRepository, cacheStore *cache.Cache) *RSSService {
	return &RSSService{repo: repo, cache: cacheStore}
}

// GetLatestPublicPostsForRSS returns the most recent public posts, capped at rssMaxItems, cached 1 hour.
func (s *RSSService) GetLatestPublicPostsForRSS() ([]RSSEntry, error) {
	if v, ok := s.cache.Get(rssCacheKey); ok {
		return v.([]RSSEntry), nil
	}
	posts, err := s.repo.GetAllPublicPostSeqs()
	if err != nil {
		return nil, err
	}
	limit := len(posts)
	if limit > rssMaxItems {
		limit = rssMaxItems
	}
	entries := make([]RSSEntry, limit)
	for i := 0; i < limit; i++ {
		p := posts[i]
		entries[i] = RSSEntry{
			SeqStr:  strconv.Itoa(p.SEQ),
			PubDate: p.RegDate,
			Subject: p.Subject,
			Summary: p.Summary,
		}
	}
	s.cache.Set(rssCacheKey, entries, rssCacheTTL)
	return entries, nil
}
