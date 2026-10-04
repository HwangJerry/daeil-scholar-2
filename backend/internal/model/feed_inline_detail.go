// feed_inline_detail.go — Expanded-post fields a feed item carries when the client asks for include=detail
package model

// NoticeInlineDetail is what an app shows when it expands a feed post in place.
// It is attached to a NoticeItem only for `include=detail`; when nil, the
// embedding item serializes exactly as before.
type NoticeInlineDetail struct {
	// Contents is the raw CONTENTS column; the presenter decodes it into ContentHtml.
	Contents        string       `json:"-"`
	ContentHtml     string       `json:"contentHtml"`
	ContentFormat   string       `json:"contentFormat"`
	Files           []FileRecord `json:"files"`
	Comments        []Comment    `json:"comments"`
	CommentsHasMore bool         `json:"commentsHasMore"`
}

// NoticeBody is the raw body of one post, read in a batch for a feed page.
type NoticeBody struct {
	SEQ           int    `db:"SEQ"`
	Contents      string `db:"CONTENTS"`
	ContentFormat string `db:"CONTENT_FORMAT"`
}

// NoticeLikeStats is a post's like count and whether the requesting user liked it.
type NoticeLikeStats struct {
	SEQ       int  `db:"BBS_SEQ"`
	LikeCnt   int  `db:"like_cnt"`
	UserLiked bool `db:"user_liked"`
}

// NoticeViewResponse is the API response for POST /api/feed/{seq}/view.
type NoticeViewResponse struct {
	Hit int `json:"hit"`
}
