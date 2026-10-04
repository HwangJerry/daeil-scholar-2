package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type fakeInlineDetailRepo struct {
	bodies     []model.NoticeBody
	comments   []model.Comment
	files      []model.FileRecord
	likeStats  []model.NoticeLikeStats
	commentErr error
	hits       map[int]int
	calls      []string
	seqArgs    [][]int
}

func (f *fakeInlineDetailRepo) GetNoticeBodies(seqs []int) ([]model.NoticeBody, error) {
	f.calls, f.seqArgs = append(f.calls, "bodies"), append(f.seqArgs, seqs)
	return f.bodies, nil
}
func (f *fakeInlineDetailRepo) GetCommentsByPosts(seqs []int) ([]model.Comment, error) {
	f.calls = append(f.calls, "comments")
	return f.comments, f.commentErr
}
func (f *fakeInlineDetailRepo) GetFilesByPosts(seqs []int) ([]model.FileRecord, error) {
	f.calls = append(f.calls, "files")
	return f.files, nil
}
func (f *fakeInlineDetailRepo) GetLikeStats(seqs []int, userSeq int) ([]model.NoticeLikeStats, error) {
	f.calls = append(f.calls, "likes")
	return f.likeStats, nil
}
func (f *fakeInlineDetailRepo) GetPublishedNoticeHit(seq int) (int, bool, error) {
	f.calls = append(f.calls, "hit")
	hit, ok := f.hits[seq]
	return hit, ok, nil
}
func (f *fakeInlineDetailRepo) IncrementHit(seq int) error {
	f.calls = append(f.calls, "increment")
	f.hits[seq]++
	return nil
}

func inlineFeed(seqs ...int) *model.FeedResponse {
	feed := &model.FeedResponse{}
	for _, seq := range seqs {
		feed.Items = append(feed.Items, model.FeedItem{Type: "notice", NoticeItem: &model.NoticeItem{SEQ: seq, CommentCnt: 99}})
	}
	return feed
}

func TestAttachToFeedBatchesAndGroupsPerPost(t *testing.T) {
	repo := &fakeInlineDetailRepo{
		bodies: []model.NoticeBody{{SEQ: 12, Contents: "PHA+", ContentFormat: "MARKDOWN"}, {SEQ: 11, Contents: "<p>x</p>", ContentFormat: "LEGACY"}},
		comments: []model.Comment{
			{BCSeq: 9, JoinSeq: 11}, {BCSeq: 8, JoinSeq: 12}, {BCSeq: 4, JoinSeq: 12},
		},
		files: []model.FileRecord{{FSeq: 1, FJoinSeq: 12}},
	}
	feed := inlineFeed(12, 11, 10)
	if err := NewFeedInlineDetailService(repo).AttachToFeed(feed); err != nil {
		t.Fatal(err)
	}
	if strings.Join(repo.calls, ",") != "bodies,comments,files" || len(repo.seqArgs[0]) != 3 {
		t.Fatalf("a page must cost exactly three batched reads: %v %v", repo.calls, repo.seqArgs)
	}
	first, second, third := feed.Items[0].NoticeItem, feed.Items[1].NoticeItem, feed.Items[2].NoticeItem
	if first.Contents != "PHA+" || first.ContentFormat != "MARKDOWN" || len(first.Files) != 1 ||
		len(first.Comments) != 2 || first.Comments[0].BCSeq != 8 || first.CommentCnt != 2 || first.CommentsHasMore {
		t.Fatalf("first = %+v", first.NoticeInlineDetail)
	}
	if second.CommentCnt != 1 || len(second.Files) != 0 || second.Files == nil {
		t.Fatalf("second = %+v", second.NoticeInlineDetail)
	}
	if third.ContentFormat != "LEGACY" || third.Comments == nil || third.Files == nil || third.CommentCnt != 0 {
		t.Fatalf("a post without a body row still gets empty arrays: %+v", third.NoticeInlineDetail)
	}
	body, err := json.Marshal(feed.Items[2])
	if err != nil || !strings.Contains(string(body), `"files":[]`) || !strings.Contains(string(body), `"comments":[]`) ||
		!strings.Contains(string(body), `"commentsHasMore":false`) || strings.Contains(string(body), "Contents") {
		t.Fatalf("serialized item = %s, %v", body, err)
	}
}

func TestAttachToFeedSkipsAnEmptyPageAndPropagatesErrors(t *testing.T) {
	repo := &fakeInlineDetailRepo{}
	if err := NewFeedInlineDetailService(repo).AttachToFeed(&model.FeedResponse{}); err != nil || len(repo.calls) != 0 {
		t.Fatalf("empty page: %v %v", err, repo.calls)
	}
	repo.commentErr = errors.New("boom")
	if err := NewFeedInlineDetailService(repo).AttachToFeed(inlineFeed(1)); err == nil {
		t.Fatal("expected the comment read error")
	}
}

func TestHeroWithDetailCopiesAndRefreshesLikes(t *testing.T) {
	repo := &fakeInlineDetailRepo{
		likeStats: []model.NoticeLikeStats{{SEQ: 5, LikeCnt: 4, UserLiked: true}},
		comments:  []model.Comment{{BCSeq: 1, JoinSeq: 5}},
	}
	cached := &model.NoticeItem{SEQ: 5, LikeCnt: 1, CommentCnt: 0}
	hero, err := NewFeedInlineDetailService(repo).HeroWithDetail(cached, 7)
	if err != nil {
		t.Fatal(err)
	}
	if hero == cached || cached.NoticeInlineDetail != nil || cached.LikeCnt != 1 {
		t.Fatalf("the cached hero must not be mutated: %+v", cached)
	}
	if hero.LikeCnt != 4 || !hero.UserLiked || hero.CommentCnt != 1 || len(hero.Comments) != 1 {
		t.Fatalf("hero = %+v", hero)
	}

	repo.likeStats = nil
	hero, err = NewFeedInlineDetailService(repo).HeroWithDetail(&model.NoticeItem{SEQ: 5, LikeCnt: 3, UserLiked: true}, 0)
	if err != nil || hero.LikeCnt != 0 || hero.UserLiked {
		t.Fatalf("a hero without likes = %+v, %v", hero, err)
	}
}

func TestRecordViewIncrementsPublishedPostsOnly(t *testing.T) {
	repo := &fakeInlineDetailRepo{hits: map[int]int{5: 41}}
	svc := NewFeedInlineDetailService(repo)
	hit, found, err := svc.RecordView(5)
	if err != nil || !found || hit != 42 || strings.Join(repo.calls, ",") != "hit,increment,hit" {
		t.Fatalf("view = %d, %v, %v (%v)", hit, found, err, repo.calls)
	}
	repo.calls = nil
	if hit, found, err := svc.RecordView(6); err != nil || found || hit != 0 || strings.Join(repo.calls, ",") != "hit" {
		t.Fatalf("missing view = %d, %v, %v (%v)", hit, found, err, repo.calls)
	}
}
