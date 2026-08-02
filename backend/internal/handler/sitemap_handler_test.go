package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
)

func TestSitemapContainsOnlyPublicMVPStaticRoutes(t *testing.T) {
	cacheStore := cache.New(0, 0)
	cacheStore.Set("sitemap:posts", []service.SitemapEntry{}, cache.NoExpiration)
	handler := NewSitemapHandler(service.NewSitemapService(nil, cacheStore), "https://example.com")

	recorder := httptest.NewRecorder()
	handler.GetSitemap(recorder, httptest.NewRequest("GET", "/sitemap.xml", nil))

	body := recorder.Body.String()
	for _, path := range []string{
		"/",
		"/about",
		"/greetings",
		"/vision",
		"/history",
		"/organization",
		"/business",
		"/disclosure",
	} {
		if !strings.Contains(body, "<loc>https://example.com"+path+"</loc>") {
			t.Fatalf("sitemap missing public route %q: %s", path, body)
		}
	}

	for _, forbidden := range []string{
		"/donation",
		"/alumni",
		"/messages",
		"/login",
		"/me",
		"/ad/",
	} {
		if strings.Contains(body, "<loc>https://example.com"+forbidden) {
			t.Fatalf("sitemap contains forbidden route %q: %s", forbidden, body)
		}
	}
}
