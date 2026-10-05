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

// FormatFeedInlineDetail decodes the body of every feed item that carries
// inline detail, through the same pipeline as FormatNoticeDetail.
func (p *FeedPresenter) FormatFeedInlineDetail(feed *model.FeedResponse) *model.FeedResponse {
	for _, item := range feed.Items {
		if item.NoticeItem != nil {
			p.FormatNoticeInlineDetail(item.NoticeItem)
		}
	}
	return feed
}

// FormatNoticeInlineDetail decodes one item's inline body into ContentHtml.
func (p *FeedPresenter) FormatNoticeInlineDetail(item *model.NoticeItem) *model.NoticeItem {
	if item.NoticeInlineDetail != nil {
		item.ContentHtml = service.DecodeContent(item.Contents, item.ContentFormat)
	}
	return item
}
