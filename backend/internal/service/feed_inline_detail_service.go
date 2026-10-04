// feed_inline_detail_service.go — Attaches expanded-post detail to feed items (include=detail) and records post views
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// FeedInlineDetailService lets apps expand feed posts client-side: it batches
// each page's bodies, comments and files (no N+1, no HIT increment) and owns
// the explicit view counter apps call on first expand.
type FeedInlineDetailService struct {
	repo repository.FeedInlineDetailQuerier
}

// NewFeedInlineDetailService creates a FeedInlineDetailService.
func NewFeedInlineDetailService(repo repository.FeedInlineDetailQuerier) *FeedInlineDetailService {
	return &FeedInlineDetailService{repo: repo}
}

// AttachToFeed adds inline detail to every notice item of a feed page. The
// page query already computed likeCnt/userLiked for this user, so only bodies,
// comments and files are read (3 queries per page, whatever its size).
func (s *FeedInlineDetailService) AttachToFeed(feed *model.FeedResponse) error {
	items := make([]*model.NoticeItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item.NoticeItem != nil {
			items = append(items, item.NoticeItem)
		}
	}
	return s.attach(items)
}

// HeroWithDetail returns a copy of the (possibly cached, user-independent) hero
// with inline detail and fresh like stats for userSeq. The cached hero is never
// mutated.
func (s *FeedInlineDetailService) HeroWithDetail(hero *model.NoticeItem, userSeq int) (*model.NoticeItem, error) {
	item := *hero
	stats, err := s.repo.GetLikeStats([]int{item.SEQ}, userSeq)
	if err != nil {
		return nil, err
	}
	item.LikeCnt, item.UserLiked = 0, false
	for _, stat := range stats {
		if stat.SEQ == item.SEQ {
			item.LikeCnt, item.UserLiked = stat.LikeCnt, stat.UserLiked
		}
	}
	if err := s.attach([]*model.NoticeItem{&item}); err != nil {
		return nil, err
	}
	return &item, nil
}

// RecordView increments a published post's HIT exactly as GET /api/feed/{seq}
// does and returns the new count. found is false for a missing or unpublished
// post.
func (s *FeedInlineDetailService) RecordView(seq int) (hit int, found bool, err error) {
	if _, found, err = s.repo.GetPublishedNoticeHit(seq); err != nil || !found {
		return 0, found, err
	}
	if err = s.repo.IncrementHit(seq); err != nil {
		return 0, true, err
	}
	hit, found, err = s.repo.GetPublishedNoticeHit(seq)
	return hit, found, err
}

// attach reads the page's bodies, comments and files in one query each and
// hangs them on the items. commentCnt is set from the comments actually
// returned, so the two always agree (the comments endpoint is not paginated,
// hence commentsHasMore is always false).
func (s *FeedInlineDetailService) attach(items []*model.NoticeItem) error {
	if len(items) == 0 {
		return nil
	}
	seqs := make([]int, len(items))
	for i, item := range items {
		seqs[i] = item.SEQ
	}
	bodies, err := s.repo.GetNoticeBodies(seqs)
	if err != nil {
		return err
	}
	comments, err := s.repo.GetCommentsByPosts(seqs)
	if err != nil {
		return err
	}
	files, err := s.repo.GetFilesByPosts(seqs)
	if err != nil {
		return err
	}

	bodyBySeq := make(map[int]model.NoticeBody, len(bodies))
	for _, body := range bodies {
		bodyBySeq[body.SEQ] = body
	}
	commentsBySeq := make(map[int][]model.Comment, len(items))
	for _, comment := range comments {
		commentsBySeq[comment.JoinSeq] = append(commentsBySeq[comment.JoinSeq], comment)
	}
	filesBySeq := make(map[int][]model.FileRecord, len(items))
	for _, file := range files {
		filesBySeq[file.FJoinSeq] = append(filesBySeq[file.FJoinSeq], file)
	}

	for _, item := range items {
		detail := &model.NoticeInlineDetail{
			ContentFormat: "LEGACY",
			Files:         make([]model.FileRecord, 0),
			Comments:      make([]model.Comment, 0),
		}
		if body, ok := bodyBySeq[item.SEQ]; ok {
			detail.Contents, detail.ContentFormat = body.Contents, body.ContentFormat
		}
		detail.Files = append(detail.Files, filesBySeq[item.SEQ]...)
		detail.Comments = append(detail.Comments, commentsBySeq[item.SEQ]...)
		item.CommentCnt = len(detail.Comments)
		item.NoticeInlineDetail = detail
	}
	return nil
}
