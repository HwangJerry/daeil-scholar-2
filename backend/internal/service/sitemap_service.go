// sitemap_service.go — Service for generating sitemap entries from public posts
package service

import (
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

const (
	sitemapCacheKey = "sitemap:posts"
	sitemapCacheTTL = 1 * time.Hour
)

// SitemapEntry is the view type for sitemap handler.
type SitemapEntry struct {
	SeqStr     string
	RegDateISO string
}

// SitemapService provides cached access to public post data for XML sitemap generation.
type SitemapService struct {
	repo  *repository.FeedRepository
	cache *cache.Cache
}

// NewSitemapService creates a new SitemapService.
func NewSitemapService(repo *repository.FeedRepository, cacheStore *cache.Cache) *SitemapService {
	return &SitemapService{repo: repo, cache: cacheStore}
}

// GetAllPublicPostsForSitemap returns all public post entries, cached 1 hour.
func (s *SitemapService) GetAllPublicPostsForSitemap() ([]SitemapEntry, error) {
	if v, ok := s.cache.Get(sitemapCacheKey); ok {
		return v.([]SitemapEntry), nil
	}
	posts, err := s.repo.GetAllPublicPostSeqs()
	if err != nil {
		return nil, err
	}
	entries := make([]SitemapEntry, len(posts))
	for i, p := range posts {
		entries[i] = SitemapEntry{
			SeqStr:     strconv.Itoa(p.SEQ),
			RegDateISO: p.RegDateISO,
		}
	}
	s.cache.Set(sitemapCacheKey, entries, sitemapCacheTTL)
	return entries, nil
}
