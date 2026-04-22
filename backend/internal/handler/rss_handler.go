// rss_handler.go — Generates an RSS 2.0 feed of public posts for Naver Search Advisor and other readers
package handler

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/service"
)

const (
	rssChannelTitle = "대일외고 장학회"
	rssChannelDesc  = "대일외고 장학회 공지 및 소식 — 대일외국어고등학교(대일외고) 재학생 장학금 지원과 동문 네트워크."
	rssLanguage     = "ko"
)

// RSSHandler serves an RSS 2.0 feed generated from public posts.
type RSSHandler struct {
	rssSvc      *service.RSSService
	siteBaseURL string
}

// NewRSSHandler creates a new RSSHandler.
func NewRSSHandler(rssSvc *service.RSSService, siteBaseURL string) *RSSHandler {
	return &RSSHandler{rssSvc: rssSvc, siteBaseURL: siteBaseURL}
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Description string  `xml:"description"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// GetRSS writes the RSS 2.0 feed response.
func (h *RSSHandler) GetRSS(w http.ResponseWriter, r *http.Request) {
	entries, err := h.rssSvc.GetLatestPublicPostsForRSS()
	if err != nil {
		http.Error(w, "rss error", http.StatusInternalServerError)
		return
	}

	items := make([]rssItem, len(entries))
	for i, e := range entries {
		link := h.siteBaseURL + "/post/" + e.SeqStr
		items[i] = rssItem{
			Title:       e.Subject,
			Link:        link,
			GUID:        rssGUID{IsPermaLink: "true", Value: link},
			PubDate:     e.PubDate.Format(time.RFC1123Z),
			Description: e.Summary,
		}
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         rssChannelTitle,
			Link:          h.siteBaseURL,
			Description:   rssChannelDesc,
			Language:      rssLanguage,
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		http.Error(w, "xml error", http.StatusInternalServerError)
	}
}
