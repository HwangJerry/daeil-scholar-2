// og_handler.go — Serves bot-facing HTML with OG meta tags for social sharing
package handler

import (
	_ "embed"
	"html/template"
	"net/http"
	"strconv"

	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/og_post.html
var ogPostTmpl string

const (
	ogSiteName    = "대일외고 장학회"
	ogDefaultDesc = "대일외고 동문 장학회 — 재학생 장학금 지원과 동문 네트워크를 운영합니다."
)

// OGHandler serves server-side OG tag HTML for bots (Kakao, Facebook, etc.).
type OGHandler struct {
	ogSvc       *service.OGService
	siteBaseURL string
	tmpl        *template.Template
}

// NewOGHandler creates a new OGHandler with the embedded template parsed.
func NewOGHandler(ogSvc *service.OGService, siteBaseURL string) *OGHandler {
	tmpl := template.Must(template.New("og_post").Parse(ogPostTmpl))
	return &OGHandler{ogSvc: ogSvc, siteBaseURL: siteBaseURL, tmpl: tmpl}
}

type ogTemplateData struct {
	Title       string
	Description string
	URL         string
	Image       string
	SiteName    string
}

// GetPostOG renders an HTML page with OG meta tags for the given post.
func (h *OGHandler) GetPostOG(w http.ResponseWriter, r *http.Request) {
	seqStr := chi.URLParam(r, "seq")
	seq, err := strconv.Atoi(seqStr)
	if err != nil || seq <= 0 {
		http.NotFound(w, r)
		return
	}

	og, err := h.ogSvc.GetOGData(seq)
	if err != nil || og == nil {
		http.NotFound(w, r)
		return
	}

	canonicalURL := h.siteBaseURL + "/post/" + seqStr
	image := h.siteBaseURL + "/logo.png"
	if og.ThumbnailURL != "" {
		image = og.ThumbnailURL
	}
	description := og.Summary
	if description == "" {
		description = ogDefaultDesc
	}

	data := ogTemplateData{
		Title:       og.Subject + " — " + ogSiteName,
		Description: description,
		URL:         canonicalURL,
		Image:       image,
		SiteName:    ogSiteName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
