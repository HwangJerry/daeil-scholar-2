package model

type BannerAdImage struct {
	BNISeq    int    `db:"BNI_SEQ" json:"bniSeq"`
	BNSeq     int    `db:"BN_SEQ" json:"bnSeq"`
	ImageURL  string `db:"IMAGE_URL" json:"imageUrl"`
	SortOrder int    `db:"SORT_ORDER" json:"sortOrder"`
}

type BannerAd struct {
	BNSeq       int             `db:"BN_SEQ" json:"bnSeq"`
	BNName      string          `db:"BN_NAME" json:"bnName"`
	BNURL       string          `db:"BN_URL" json:"bnUrl"`
	BNStartDate *string         `db:"BN_START_DATE" json:"bnStartDate"`
	BNEndDate   *string         `db:"BN_END_DATE" json:"bnEndDate"`
	Images      []BannerAdImage `json:"images"`
}

type AdminBannerAdRow struct {
	BNSeq       int             `db:"BN_SEQ" json:"bnSeq"`
	BNName      string          `db:"BN_NAME" json:"bnName"`
	BNURL       string          `db:"BN_URL" json:"bnUrl"`
	OpenYN      string          `db:"OPEN_YN" json:"openYn"`
	Indx        int             `db:"INDX" json:"indx"`
	BNStartDate *string         `db:"BN_START_DATE" json:"bnStartDate"`
	BNEndDate   *string         `db:"BN_END_DATE" json:"bnEndDate"`
	CreatedAt   string          `db:"CREATED_AT" json:"createdAt"`
	UpdatedAt   string          `db:"UPDATED_AT" json:"updatedAt"`
	Images      []BannerAdImage `json:"images"`
	ViewCount   int             `db:"VIEW_COUNT" json:"viewCount"`
	ClickCount  int             `db:"CLICK_COUNT" json:"clickCount"`
}

type AdminBannerAdInsert struct {
	BNName      string
	BNURL       string
	OpenYN      string
	Indx        int
	BNStartDate *string
	BNEndDate   *string
	ImageURLs   []string
}

type AdminBannerAdStats struct {
	BNSeq      int `db:"BN_SEQ" json:"bnSeq"`
	ViewCount  int `db:"VIEW_COUNT" json:"viewCount"`
	ClickCount int `db:"CLICK_COUNT" json:"clickCount"`
}

type BannerAdLog struct {
	BNLSeq    int    `db:"BNL_SEQ"`
	BNSeq     int    `db:"BN_SEQ"`
	LogType   string `db:"LOG_TYPE"`
	CreatedAt string `db:"CREATED_AT"`
}
