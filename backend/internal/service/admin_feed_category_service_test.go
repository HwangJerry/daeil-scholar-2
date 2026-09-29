// admin_feed_category_service_test.go — Unit tests for feed category name, count, default, delete and reorder rules
package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

// fakeFeedCategoryStore is an in-memory AdminFeedCategoryStore. postCounts
// stands in for the NOTICE posts referencing each category.
type fakeFeedCategoryStore struct {
	cats       []model.AdminFeedCategory
	postCounts map[int]int
	nextSeq    int
	reordered  []int
	deleted    []int
	movedTo    map[int]int
}

func newFakeFeedCategoryStore() *fakeFeedCategoryStore {
	return &fakeFeedCategoryStore{
		cats: []model.AdminFeedCategory{
			{Seq: 1, Code: "notice", Name: "공지", SortOrder: 1, OpenYN: "Y", IsDefault: "Y"},
			{Seq: 2, Code: "etc", Name: "기타", SortOrder: 2, OpenYN: "Y", IsDefault: "N"},
		},
		postCounts: map[int]int{},
		nextSeq:    3,
		movedTo:    map[int]int{},
	}
}

func (f *fakeFeedCategoryStore) GetAll() ([]model.AdminFeedCategory, error) {
	return append([]model.AdminFeedCategory(nil), f.cats...), nil
}

func (f *fakeFeedCategoryStore) GetBySeq(seq int) (*model.AdminFeedCategory, error) {
	for i := range f.cats {
		if f.cats[i].Seq == seq {
			cat := f.cats[i]
			return &cat, nil
		}
	}
	return nil, nil
}

func (f *fakeFeedCategoryStore) Insert(name, openYN string) (int, error) {
	seq := f.nextSeq
	f.nextSeq++
	f.cats = append(f.cats, model.AdminFeedCategory{Seq: seq, Code: "c" + string(rune('0'+seq)), Name: name, SortOrder: len(f.cats) + 1, OpenYN: openYN, IsDefault: "N"})
	return seq, nil
}

func (f *fakeFeedCategoryStore) Update(seq int, name, openYN string) error {
	for i := range f.cats {
		if f.cats[i].Seq == seq {
			f.cats[i].Name, f.cats[i].OpenYN = name, openYN
		}
	}
	return nil
}

func (f *fakeFeedCategoryStore) Reorder(seqs []int) error {
	f.reordered = seqs
	return nil
}

func (f *fakeFeedCategoryStore) Delete(seq int, moveToSeq *int) (int, error) {
	if moveToSeq == nil && f.postCounts[seq] > 0 {
		return f.postCounts[seq], nil
	}
	if moveToSeq != nil {
		f.movedTo[seq] = *moveToSeq
	}
	f.deleted = append(f.deleted, seq)
	return 0, nil
}

func newTestFeedCategoryService(store *fakeFeedCategoryStore) *AdminFeedCategoryService {
	return NewAdminFeedCategoryService(store, cache.New(time.Minute, time.Minute))
}

func assertRuleError(t *testing.T, err error, want *FeedCategoryRuleError) {
	t.Helper()
	var rule *FeedCategoryRuleError
	if !errors.As(err, &rule) || rule != want {
		t.Fatalf("err = %v, want %s", err, want.Code)
	}
}

func TestFeedCategoryCreateValidatesTrimmedNameLength(t *testing.T) {
	for _, name := range []string{"", "   ", "아홉글자카테고리임", strings.Repeat("a", 9)} {
		store := newFakeFeedCategoryStore()
		_, err := newTestFeedCategoryService(store).Create(model.AdminFeedCategoryUpsert{Name: name})
		assertRuleError(t, err, ErrFeedCategoryNameInvalid)
		if len(store.cats) != 2 {
			t.Fatalf("name %q must not be stored", name)
		}
	}
	store := newFakeFeedCategoryStore()
	cat, err := newTestFeedCategoryService(store).Create(model.AdminFeedCategoryUpsert{Name: "  여덟글자카테고리 "})
	if err != nil {
		t.Fatalf("an 8-rune name must pass: %v", err)
	}
	if cat.Name != "여덟글자카테고리" || cat.OpenYN != "Y" {
		t.Fatalf("created = %+v; want the trimmed name, open by default", cat)
	}
}

func TestFeedCategoryNamesAreUniqueIgnoringCase(t *testing.T) {
	store := newFakeFeedCategoryStore()
	svc := newTestFeedCategoryService(store)
	if _, err := svc.Create(model.AdminFeedCategoryUpsert{Name: "News"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(model.AdminFeedCategoryUpsert{Name: " news "})
	assertRuleError(t, err, ErrFeedCategoryNameDuplicate)
	_, err = svc.Update(2, model.AdminFeedCategoryUpsert{Name: "공지", OpenYN: "Y"})
	assertRuleError(t, err, ErrFeedCategoryNameDuplicate)
	// Keeping one's own name (a visibility-only change) is not a clash.
	if _, err := svc.Update(2, model.AdminFeedCategoryUpsert{Name: "기타", OpenYN: "N"}); err != nil {
		t.Fatalf("own name must be allowed: %v", err)
	}
}

func TestFeedCategoryCreateStopsAtSix(t *testing.T) {
	store := newFakeFeedCategoryStore()
	svc := newTestFeedCategoryService(store)
	for _, name := range []string{"장학", "동문", "행사", "모집"} {
		if _, err := svc.Create(model.AdminFeedCategoryUpsert{Name: name, OpenYN: "Y"}); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	_, err := svc.Create(model.AdminFeedCategoryUpsert{Name: "일곱째", OpenYN: "Y"})
	assertRuleError(t, err, ErrFeedCategoryLimit)
	if len(store.cats) != model.FeedCategoryMaxCount {
		t.Fatalf("stored %d categories, want %d", len(store.cats), model.FeedCategoryMaxCount)
	}
}

func TestFeedCategoryDefaultCanBeRenamedButNotHiddenOrDeleted(t *testing.T) {
	store := newFakeFeedCategoryStore()
	svc := newTestFeedCategoryService(store)
	_, err := svc.Update(1, model.AdminFeedCategoryUpsert{Name: "공지", OpenYN: "N"})
	assertRuleError(t, err, ErrFeedCategoryDefaultHide)
	assertRuleError(t, svc.Delete(1, nil), ErrFeedCategoryDefaultDelete)
	moveTo := 2
	assertRuleError(t, svc.Delete(1, &moveTo), ErrFeedCategoryDefaultDelete)
	renamed, err := svc.Update(1, model.AdminFeedCategoryUpsert{Name: "새 소식", OpenYN: "Y"})
	if err != nil || renamed.Name != "새 소식" || renamed.OpenYN != "Y" {
		t.Fatalf("rename = %+v, %v", renamed, err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("default was deleted: %v", store.deleted)
	}
}

func TestFeedCategoryUpdateRejectsUnknownSeqAndOpenValue(t *testing.T) {
	svc := newTestFeedCategoryService(newFakeFeedCategoryStore())
	if _, err := svc.Update(99, model.AdminFeedCategoryUpsert{Name: "없음"}); !errors.Is(err, ErrFeedCategoryNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	_, err := svc.Update(2, model.AdminFeedCategoryUpsert{Name: "기타", OpenYN: "X"})
	assertRuleError(t, err, ErrFeedCategoryOpenInvalid)
}

func TestFeedCategoryDeleteWithPostsRequiresAnotherExistingCategory(t *testing.T) {
	store := newFakeFeedCategoryStore()
	store.postCounts[2] = 4
	svc := newTestFeedCategoryService(store)

	var hasPosts *FeedCategoryHasPostsError
	if err := svc.Delete(2, nil); !errors.As(err, &hasPosts) || hasPosts.PostCount != 4 {
		t.Fatalf("err = %v, want has-posts with count 4", err)
	}
	self, unknown := 2, 99
	assertRuleError(t, svc.Delete(2, &self), ErrFeedCategoryMoveInvalid)
	assertRuleError(t, svc.Delete(2, &unknown), ErrFeedCategoryMoveInvalid)
	if len(store.deleted) != 0 {
		t.Fatalf("refused deletes must not reach the store: %v", store.deleted)
	}

	toDefault := 1
	if err := svc.Delete(2, &toDefault); err != nil {
		t.Fatal(err)
	}
	if store.movedTo[2] != 1 || len(store.deleted) != 1 {
		t.Fatalf("posts must move to 1 and the category be deleted: moved=%v deleted=%v", store.movedTo, store.deleted)
	}
}

func TestFeedCategoryDeleteWithoutPostsNeedsNoTarget(t *testing.T) {
	store := newFakeFeedCategoryStore()
	if err := newTestFeedCategoryService(store).Delete(2, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != 2 {
		t.Fatalf("deleted = %v", store.deleted)
	}
	if err := newTestFeedCategoryService(store).Delete(42, nil); !errors.Is(err, ErrFeedCategoryNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestFeedCategoryReorderNeedsEverySeqOnce(t *testing.T) {
	store := newFakeFeedCategoryStore()
	svc := newTestFeedCategoryService(store)
	for _, seqs := range [][]int{nil, {2}, {2, 2}, {2, 3}, {1, 2, 3}} {
		assertRuleError(t, svc.Reorder(seqs), ErrFeedCategoryOrderInvalid)
	}
	if store.reordered != nil {
		t.Fatalf("invalid orders must not reach the store: %v", store.reordered)
	}
	if err := svc.Reorder([]int{2, 1}); err != nil {
		t.Fatal(err)
	}
	if len(store.reordered) != 2 || store.reordered[0] != 2 || store.reordered[1] != 1 {
		t.Fatalf("reordered = %v", store.reordered)
	}
}

func TestFeedCategoryChangesDropTheCachedHero(t *testing.T) {
	store := newFakeFeedCategoryStore()
	svc := newTestFeedCategoryService(store)
	svc.cache.Set(feedHeroCacheKey, &model.NoticeItem{SEQ: 1, CategoryName: "공지"}, time.Minute)
	if _, err := svc.Update(1, model.AdminFeedCategoryUpsert{Name: "소식"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.cache.Get(feedHeroCacheKey); ok {
		t.Fatal("a rename must not leave the hero showing the old category name")
	}
}
