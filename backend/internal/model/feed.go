package model

// FeedItem represents a single item in the feed (notice or ad).
type FeedItem struct {
	Type        string `json:"type"`
	*NoticeItem `json:",omitempty"`
	*AdItem     `json:",omitempty"`
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
	LikeCnt      int    `json:"likeCnt"`
	CommentCnt   int    `json:"commentCnt"`
	IsPinned     string `db:"IS_PINNED" json:"isPinned,omitempty"`
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
	LikeCnt       int    `json:"likeCnt"`
	CommentCnt    int    `json:"commentCnt"`
	UserLiked     bool   `json:"userLiked"`
	Files         []FileRecord   `json:"files,omitempty"`
	ContentHtml   string `json:"contentHtml"`
	ContentMd     string `json:"contentMd,omitempty"`
}

// Comment represents a row in WEO_BOARDCOMAND table.
type Comment struct {
	BCSeq    int    `db:"BC_SEQ" json:"bcSeq"`
	JoinSeq  int    `db:"JOIN_SEQ" json:"joinSeq"`
	USRSeq   int    `db:"USR_SEQ" json:"usrSeq"`
	RegName  string `db:"REG_NAME" json:"regName"`
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

// AdItem represents an ad from MAIN_AD table.
type AdItem struct {
	MASeq      int    `db:"MA_SEQ" json:"maSeq"`
	MAName     string `db:"MA_NAME" json:"maName"`
	MAURL      string `db:"MA_URL" json:"maUrl"`
	ImageURL   string `db:"MA_IMG" json:"imageUrl"`
	AdTier     string `db:"AD_TIER" json:"adTier"`
	TitleLabel string `db:"AD_TITLE_LABEL" json:"titleLabel"`
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
