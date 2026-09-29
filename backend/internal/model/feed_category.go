// feed_category.go — Models for admin-managed feed post categories (ALUMNI_FEED_CATEGORY)
package model

// FeedCategoryMaxCount is the most categories an operator can keep; apps show
// one tab per open category next to 전체.
const FeedCategoryMaxCount = 6

// FeedCategoryNameMaxRunes bounds a trimmed category name (Unicode code points).
const FeedCategoryNameMaxRunes = 8

// FeedCategory is one app feed tab in GET /api/feed `categories`.
type FeedCategory struct {
	Code string `db:"FC_CODE" json:"code"`
	Name string `db:"FC_NAME" json:"name"`
}

// AdminFeedCategory is one row of GET /api/admin/feed-categories. PostCount
// counts NOTICE posts, including soft-deleted ones the admin list still shows;
// the default category also counts posts with no (or a dangling) category.
type AdminFeedCategory struct {
	Seq       int    `db:"FC_SEQ" json:"seq"`
	Code      string `db:"FC_CODE" json:"code"`
	Name      string `db:"FC_NAME" json:"name"`
	SortOrder int    `db:"SORT_ORDER" json:"sortOrder"`
	OpenYN    string `db:"OPEN_YN" json:"openYn"`
	IsDefault string `db:"IS_DEFAULT" json:"isDefault"`
	PostCount int    `db:"post_count" json:"postCount"`
}

// AdminFeedCategoryUpsert is the create/update body.
type AdminFeedCategoryUpsert struct {
	Name   string `json:"name"`
	OpenYN string `json:"openYn"`
}

// AdminFeedCategoryReorderRequest carries every category seq in the new order.
type AdminFeedCategoryReorderRequest struct {
	Seqs []int `json:"seqs"`
}

// AdminFeedCategoryDeleteRequest optionally names where the category's posts move.
type AdminFeedCategoryDeleteRequest struct {
	MoveToSeq *int `json:"moveToSeq"`
}
