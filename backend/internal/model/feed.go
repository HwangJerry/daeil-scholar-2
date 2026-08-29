package model

import (
	"encoding/json"
	"time"
)

// FeedItem represents a single notice item in the feed.
type FeedItem struct {
	Type        string `json:"type"`
	*NoticeItem `json:",omitempty"`
}

// MarshalJSON preserves the feed item type alongside the embedded notice.
func (f FeedItem) MarshalJSON() ([]byte, error) {
	if f.NoticeItem != nil {
		return json.Marshal(struct {
			Type string `json:"type"`
			*NoticeItem
		}{Type: f.Type, NoticeItem: f.NoticeItem})
	}
	return json.Marshal(struct {
		Type string `json:"type"`
	}{Type: f.Type})
}

// NoticeItem represents a notice post from WEO_BOARDBBS.
type NoticeItem struct {
	SEQ          int    `db:"SEQ" json:"seq"`
	Subject      string `db:"SUBJECT" json:"subject"`
	Summary      string `db:"SUMMARY" json:"summary"`
	ThumbnailURL string `db:"THUMBNAIL_URL" json:"thumbnailUrl"`
	RegDate      string `db:"REG_DATE" json:"regDate"`
	RegName      string `db:"REG_NAME" json:"regName"`
	Hit          int    `db:"HIT" json:"hit"`
	LikeCnt      int    `db:"like_cnt" json:"likeCnt"`
	CommentCnt   int    `db:"comment_cnt" json:"commentCnt"`
	IsPinned     string `db:"IS_PINNED" json:"isPinned,omitempty"`
	UserLiked    bool   `db:"user_liked" json:"userLiked"`
}

// NoticeDetail is the full detail of a notice post (DB scan target).
type NoticeDetail struct {
	SEQ           int    `db:"SEQ" json:"seq"`
	Subject       string `db:"SUBJECT" json:"subject"`
	Contents      string `db:"CONTENTS" json:"-"`
	ContentsMD    string `db:"CONTENTS_MD" json:"-"`
	ContentFormat string `db:"CONTENT_FORMAT" json:"contentFormat"`
	Summary       string `db:"SUMMARY" json:"summary"`
	ThumbnailURL  string `db:"THUMBNAIL_URL" json:"thumbnailUrl"`
	RegDate       string `db:"REG_DATE" json:"regDate"`
	RegName       string `db:"REG_NAME" json:"regName"`
	Hit           int    `db:"HIT" json:"hit"`
	IsPinned      string `db:"IS_PINNED" json:"isPinned"`
	LikeCnt       int    `json:"likeCnt"`
	CommentCnt    int    `json:"commentCnt"`
	UserLiked     bool   `json:"userLiked"`
	Files         []FileRecord   `json:"files,omitempty"`
	ContentHtml   string `json:"contentHtml"`
	ContentMd     string `json:"contentMd,omitempty"`
}

// DisclosureItem represents a public-disclosure list row from WEO_BOARDBBS (GATE='DISCLOSURE').
type DisclosureItem struct {
	SEQ             int    `db:"SEQ" json:"seq"`
	Subject         string `db:"SUBJECT" json:"subject"`
	Summary         string `db:"SUMMARY" json:"summary"`
	RegDate         string `db:"REG_DATE" json:"regDate"`
	RegName         string `db:"REG_NAME" json:"regName"`
	Hit             int    `db:"HIT" json:"hit"`
	AttachmentCount int    `db:"attachment_count" json:"attachmentCount"`
}

// DisclosureListResponse is the API response for GET /api/disclosure.
type DisclosureListResponse struct {
	Items      []DisclosureItem `json:"items"`
	NextCursor string           `json:"nextCursor"`
	HasMore    bool             `json:"hasMore"`
}

// Comment represents a row in WEO_BOARDCOMAND table.
type Comment struct {
	BCSeq    int    `db:"BC_SEQ" json:"bcSeq"`
	JoinSeq  int    `db:"JOIN_SEQ" json:"joinSeq"`
	USRSeq   int    `db:"USR_SEQ" json:"usrSeq"`
	RegName  string `db:"NICKNAME" json:"regName"`
	Contents string `db:"CONTENTS" json:"contents"`
	RegDate  string `db:"REG_DATE" json:"regDate"`
}

// LikeToggleResponse is the API response for POST /api/feed/{seq}/like.
type LikeToggleResponse struct {
	Liked   bool `json:"liked"`
	LikeCnt int  `json:"likeCnt"`
}

// CommentCreateRequest is the request body for POST /api/feed/{seq}/comments.
type CommentCreateRequest struct {
	Contents string `json:"contents"`
}

// FileRecord represents a row in WEO_FILES table.
type FileRecord struct {
	FSeq        int    `db:"F_SEQ" json:"fSeq"`
	FGate       string `db:"F_GATE" json:"fGate"`
	FJoinSeq    int    `db:"F_JOIN_SEQ" json:"fJoinSeq"`
	TypeName    string `db:"TYPE_NAME" json:"typeName"`
	FileName    string `db:"FILE_NAME" json:"fileName"`
	FileSize    string `db:"FILE_SIZE" json:"fileSize"`
	FilePath    string `db:"FILE_PATH" json:"filePath"`
	FileOrgName string `db:"FILE_ORG_NAME" json:"fileOrgName"`
	OpenYN      string `db:"OPEN_YN" json:"openYn"`
}

// PostSibling represents a prev/next post reference.
type PostSibling struct {
	SEQ     int    `db:"SEQ" json:"seq"`
	Subject string `db:"SUBJECT" json:"subject"`
}

// PostSiblings holds the prev and next post references for navigation.
type PostSiblings struct {
	Prev *PostSibling `json:"prev"`
	Next *PostSibling `json:"next"`
}

// FeedResponse is the API response for GET /api/feed.
type FeedResponse struct {
	Items      []FeedItem `json:"items"`
	NextCursor string     `json:"nextCursor"`
	HasMore    bool       `json:"hasMore"`
}

// OGData holds the minimum fields needed for bot OG rendering.
type OGData struct {
	Subject      string `db:"SUBJECT"`
	Summary      string `db:"SUMMARY"`
	ThumbnailURL string `db:"THUMBNAIL_URL"`
}

// SitemapPost holds fields needed for sitemap and RSS entry generation.
// REG_DATE is selected twice so RSS gets the full timestamp while sitemap keeps YYYY-MM-DD.
type SitemapPost struct {
	SEQ        int       `db:"SEQ"`
	RegDateISO string    `db:"REG_DATE"`
	RegDate    time.Time `db:"REG_DATE_RAW"`
	Subject    string    `db:"SUBJECT"`
	Summary    string    `db:"SUMMARY"`
}
