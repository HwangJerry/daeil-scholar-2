// sitemap_handler.go — Generates dynamic XML sitemap for public posts
package handler

import (
	"encoding/xml"
	"net/http"

	"github.com/dflh-saf/backend/internal/service"
)

// SitemapHandler generates the XML sitemap for public pages and posts.
type SitemapHandler struct {
	sitemapSvc  *service.SitemapService
	siteBaseURL string
}

// NewSitemapHandler creates a new SitemapHandler.
func NewSitemapHandler(sitemapSvc *service.SitemapService, siteBaseURL string) *SitemapHandler {
	return &SitemapHandler{sitemapSvc: sitemapSvc, siteBaseURL: siteBaseURL}
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

// GetSitemap writes the XML sitemap response with fixed and dynamic post URLs.
func (h *SitemapHandler) GetSitemap(w http.ResponseWriter, r *http.Request) {
	base := h.siteBaseURL
	urls := []sitemapURL{
		{Loc: base + "/", ChangeFreq: "daily", Priority: "1.0"},
		{Loc: base + "/donation", ChangeFreq: "weekly", Priority: "0.7"},
	}

	posts, err := h.sitemapSvc.GetAllPublicPostsForSitemap()
	if err == nil {
		for _, p := range posts {
			urls = append(urls, sitemapURL{
				Loc:        base + "/post/" + p.SeqStr,
				LastMod:    p.RegDateISO,
				ChangeFreq: "monthly",
				Priority:   "0.8",
			})
		}
	}

	sitemap := urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(sitemap); err != nil {
		http.Error(w, "xml error", http.StatusInternalServerError)
	}
}
