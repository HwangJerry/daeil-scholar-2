package repository

import (
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

// newFeedCategoryDB is the production baseline plus migrations 078 to 080 on
// the real MariaDB 10.1 engine, so the category joins and transactions run as
// they will in production.
func newFeedCategoryDB(t *testing.T) *sqlx.DB {
	t.Helper()
	inputs := append(mariadb.ProdBaseline(t),
		mariadb.File("../../migrations/078_create_notification_inbox_state.sql"),
		mariadb.File("../../migrations/079_create_feed_categories.sql"),
		mariadb.File("../../migrations/080_add_board_official_profile.sql"))
	return mariadb.Start(t).NewDatabase(t, inputs...).DB
}

func insertCategoryNotice(t *testing.T, db *sqlx.DB, subject string, categorySeq *int) int {
	t.Helper()
	seq, err := NewAdminNoticeRepository(db).InsertNotice(&model.AdminNoticeInsert{
		Subject: subject, IsPinned: "N", RegName: "운영자", USRSeq: 1, FeedCategorySeq: categorySeq,
	})
	if err != nil {
		t.Fatal(err)
	}
	return seq
}

func feedCategoriesBySeq(t *testing.T, repo *AdminFeedCategoryRepository) map[int]model.AdminFeedCategory {
	t.Helper()
	all, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	bySeq := make(map[int]model.AdminFeedCategory, len(all))
	for _, cat := range all {
		bySeq[cat.Seq] = cat
	}
	return bySeq
}

func TestFeedCategoryMigrationSeedsTheDefaultAndEtc(t *testing.T) {
	db := newFeedCategoryDB(t)
	all, err := NewAdminFeedCategoryRepository(db).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("seeded %d categories: %+v", len(all), all)
	}
	notice, etc := all[0], all[1]
	if notice.Code != "notice" || notice.Name != "공지" || notice.IsDefault != "Y" || notice.OpenYN != "Y" || notice.SortOrder != 1 {
		t.Fatalf("default seed = %+v", notice)
	}
	if etc.Code != "etc" || etc.Name != "기타" || etc.IsDefault != "N" || etc.SortOrder != 2 {
		t.Fatalf("etc seed = %+v", etc)
	}
}

func TestFeedCategoryInsertAppendsWithStableCode(t *testing.T) {
	db := newFeedCategoryDB(t)
	repo := NewAdminFeedCategoryRepository(db)
	seq, err := repo.Insert("장학", "N")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := repo.GetBySeq(seq)
	if err != nil || cat == nil {
		t.Fatalf("GetBySeq = %+v, %v", cat, err)
	}
	if cat.Code != "c3" || cat.Name != "장학" || cat.SortOrder != 3 || cat.OpenYN != "N" || cat.IsDefault != "N" {
		t.Fatalf("created = %+v", cat)
	}
	// A rename never changes the code apps key on.
	if err := repo.Update(seq, "장학금", "Y"); err != nil {
		t.Fatal(err)
	}
	if cat, _ := repo.GetBySeq(seq); cat.Code != "c3" || cat.Name != "장학금" || cat.OpenYN != "Y" {
		t.Fatalf("after rename = %+v", cat)
	}
}

func TestFeedCategoryReorderAndOpenList(t *testing.T) {
	db := newFeedCategoryDB(t)
	repo := NewAdminFeedCategoryRepository(db)
	hidden, err := repo.Insert("숨김", "N")
	if err != nil {
		t.Fatal(err)
	}
	events, err := repo.Insert("행사", "Y")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Reorder([]int{events, 2, hidden, 1}); err != nil {
		t.Fatal(err)
	}
	all, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, cat := range all {
		order = append(order, cat.Code)
	}
	if strings.Join(order, ",") != "c4,etc,c3,notice" {
		t.Fatalf("admin order = %v", order)
	}
	// /api/feed categories: open only, in the new order.
	open, err := NewFeedRepository(db).GetOpenFeedCategories()
	if err != nil {
		t.Fatal(err)
	}
	want := []model.FeedCategory{{Code: "c4", Name: "행사"}, {Code: "etc", Name: "기타"}, {Code: "notice", Name: "공지"}}
	if len(open) != len(want) {
		t.Fatalf("open categories = %+v", open)
	}
	for i := range want {
		if open[i] != want[i] {
			t.Fatalf("open categories = %+v, want %+v", open, want)
		}
	}
}

func TestFeedItemsResolveTheirCategory(t *testing.T) {
	db := newFeedCategoryDB(t)
	repo := NewAdminFeedCategoryRepository(db)
	hidden, err := repo.Insert("숨김", "N")
	if err != nil {
		t.Fatal(err)
	}
	etc := 2
	legacy := insertCategoryNotice(t, db, "카테고리 없음", nil)
	dangling := insertCategoryNotice(t, db, "사라진 카테고리", nil)
	db.MustExec(`UPDATE WEO_BOARDBBS SET FEED_CATEGORY_SEQ = 999 WHERE SEQ = ?`, dangling)
	inEtc := insertCategoryNotice(t, db, "기타 글", &etc)
	inHidden := insertCategoryNotice(t, db, "숨김 글", &hidden)

	feed := NewFeedRepository(db)
	items, err := feed.GetNotices(0, 10, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int][2]string{}
	for _, item := range items {
		got[item.SEQ] = [2]string{item.Category, item.CategoryName}
	}
	want := map[int][2]string{
		legacy:   {"notice", "공지"},
		dangling: {"notice", "공지"},
		inEtc:    {"etc", "기타"},
		// A hidden category's posts stay under 전체 with their own name.
		inHidden: {"c3", "숨김"},
	}
	if len(got) != len(want) {
		t.Fatalf("feed items = %v", got)
	}
	for seq, pair := range want {
		if got[seq] != pair {
			t.Fatalf("post %d category = %v, want %v", seq, got[seq], pair)
		}
	}

	hero, err := feed.GetHeroNotice()
	if err != nil || hero == nil || hero.SEQ != inHidden || hero.Category != "c3" {
		t.Fatalf("hero = %+v, %v", hero, err)
	}
	detail, err := feed.GetNoticeDetail(legacy)
	if err != nil || detail == nil || detail.Category != "notice" || detail.CategoryName != "공지" || detail.CategorySeq != 0 {
		t.Fatalf("public detail = %+v, %v", detail, err)
	}

	// Renames show at once: names are joined at read time.
	if err := repo.Update(1, "새 소식", "Y"); err != nil {
		t.Fatal(err)
	}
	detail, err = feed.GetNoticeDetail(legacy)
	if err != nil || detail.CategoryName != "새 소식" {
		t.Fatalf("renamed detail = %+v, %v", detail, err)
	}
}

func TestAdminNoticeCategoryFilterAndCounts(t *testing.T) {
	db := newFeedCategoryDB(t)
	etc := 2
	legacy := insertCategoryNotice(t, db, "카테고리 없음", nil)
	inEtc := insertCategoryNotice(t, db, "기타 글", &etc)
	deletedEtc := insertCategoryNotice(t, db, "삭제된 기타 글", &etc)
	notices := NewAdminNoticeRepository(db)
	if err := notices.DeleteNotice(deletedEtc); err != nil {
		t.Fatal(err)
	}

	rows, total, err := notices.GetNotices(1, 20, "", 1)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].SEQ != legacy || rows[0].CategorySeq != 1 || rows[0].CategoryName != "공지" {
		t.Fatalf("default filter = %+v (total %d), %v", rows, total, err)
	}
	rows, total, err = notices.GetNotices(1, 20, "", etc)
	if err != nil || total != 2 || len(rows) != 2 || rows[0].SEQ != deletedEtc || rows[1].SEQ != inEtc {
		t.Fatalf("etc filter = %+v (total %d), %v", rows, total, err)
	}
	if _, total, err = notices.GetNotices(1, 20, "", 0); err != nil || total != 3 {
		t.Fatalf("unfiltered total = %d, %v", total, err)
	}

	detail, err := notices.GetNoticeForEdit(legacy)
	if err != nil || detail.CategorySeq != 1 || detail.CategoryName != "공지" {
		t.Fatalf("admin detail = %+v, %v", detail, err)
	}
	// Legacy posts are reclassified without touching their content.
	if err := notices.UpdateNoticeCategory(legacy, etc); err != nil {
		t.Fatal(err)
	}
	counts := feedCategoriesBySeq(t, NewAdminFeedCategoryRepository(db))
	if counts[1].PostCount != 0 || counts[etc].PostCount != 3 {
		t.Fatalf("post counts = %+v", counts)
	}
	// An update without a category keeps the current one.
	if err := notices.UpdateNotice(legacy, &model.AdminNoticeInsert{Subject: "수정", IsPinned: "N"}); err != nil {
		t.Fatal(err)
	}
	if detail, _ := notices.GetNoticeForEdit(legacy); detail.CategorySeq != etc {
		t.Fatalf("update without category moved the post to %d", detail.CategorySeq)
	}
}

func TestFeedCategoryDeleteMovesPostsInOneTransaction(t *testing.T) {
	db := newFeedCategoryDB(t)
	repo := NewAdminFeedCategoryRepository(db)
	scholarship, err := repo.Insert("장학", "Y")
	if err != nil {
		t.Fatal(err)
	}
	first := insertCategoryNotice(t, db, "장학 1", &scholarship)
	second := insertCategoryNotice(t, db, "장학 2", &scholarship)
	insertCategoryNotice(t, db, "기본", nil)
	if counts := feedCategoriesBySeq(t, repo); counts[scholarship].PostCount != 2 || counts[1].PostCount != 1 {
		t.Fatalf("post counts = %+v", counts)
	}

	// Without a target the delete is refused and nothing changes.
	postCount, err := repo.Delete(scholarship, nil)
	if err != nil || postCount != 2 {
		t.Fatalf("Delete without target = %d, %v", postCount, err)
	}
	if cat, _ := repo.GetBySeq(scholarship); cat == nil {
		t.Fatal("a category with posts was deleted without a target")
	}

	// A failure after the posts moved rolls the move back too.
	db.MustExec(`CREATE TRIGGER _test_block_feed_category_delete BEFORE DELETE ON ALUMNI_FEED_CATEGORY
		FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'blocked by test'`)
	etc := 2
	if _, err := repo.Delete(scholarship, &etc); err == nil {
		t.Fatal("the blocked delete must fail")
	}
	var stillThere int
	if err := db.Get(&stillThere, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE FEED_CATEGORY_SEQ = ?`, scholarship); err != nil || stillThere != 2 {
		t.Fatalf("posts left on the category after a failed delete = %d, %v", stillThere, err)
	}
	db.MustExec(`DROP TRIGGER _test_block_feed_category_delete`)

	if _, err := repo.Delete(scholarship, &etc); err != nil {
		t.Fatal(err)
	}
	if cat, _ := repo.GetBySeq(scholarship); cat != nil {
		t.Fatalf("category still exists: %+v", cat)
	}
	var moved int
	if err := db.Get(&moved, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE FEED_CATEGORY_SEQ = ? AND SEQ IN (?, ?)`, etc, first, second); err != nil || moved != 2 {
		t.Fatalf("moved posts = %d, %v", moved, err)
	}

	// The default row is never deleted by the repository either.
	if _, err := repo.Delete(1, nil); err != nil {
		t.Fatal(err)
	}
	if cat, _ := repo.GetBySeq(1); cat == nil {
		t.Fatal("the default category was deleted")
	}
}
