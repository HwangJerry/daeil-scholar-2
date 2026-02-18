package presenter

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

// FeedPresenter transforms domain models into API response shapes.
type FeedPresenter struct{}

func NewFeedPresenter() *FeedPresenter {
	return &FeedPresenter{}
}

// FormatNoticeDetail decodes raw DB content and populates client-facing fields.
func (p *FeedPresenter) FormatNoticeDetail(detail *model.NoticeDetail) *model.NoticeDetail {
	detail.ContentHtml = service.DecodeContent(detail.Contents, detail.ContentFormat)
	if detail.ContentsMD != "" {
		detail.ContentMd = detail.ContentsMD
	}
	return detail
}

// FormatNoticeDetailForAdmin includes the original Markdown source for editing.
func (p *FeedPresenter) FormatNoticeDetailForAdmin(detail *model.NoticeDetail) *model.NoticeDetail {
	detail.ContentHtml = service.DecodeContent(detail.Contents, detail.ContentFormat)
	if detail.ContentsMD != "" {
		detail.ContentMd = detail.ContentsMD
	}
	return detail
}
