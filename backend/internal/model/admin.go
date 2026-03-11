// Admin-specific model types for CRUD and dashboard operations
package model

// --- Notice ---

type AdminNoticeRow struct {
	SEQ           int    `db:"SEQ" json:"seq"`
	Subject       string `db:"SUBJECT" json:"subject"`
	RegDate       string `db:"REG_DATE" json:"regDate"`
	RegName       string `db:"REG_NAME" json:"regName"`
	Hit           int    `db:"HIT" json:"hit"`
	OpenYN        string `db:"OPEN_YN" json:"openYn"`
	IsPinned      string `db:"IS_PINNED" json:"isPinned"`
	ContentFormat string `db:"CONTENT_FORMAT" json:"contentFormat"`
}

type AdminNoticeInsert struct {
	Subject      string
	Contents     string // Base64-encoded sanitized HTML
	ContentsMD   string // raw Markdown
	Summary      string
	ThumbnailURL string
	IsPinned     string
	RegName      string
	USRSeq       int
}

// --- Ad ---

type AdminAdRow struct {
	MASeq        int    `db:"MA_SEQ" json:"maSeq"`
	MAName       string `db:"MA_NAME" json:"maName"`
	MAURL        string `db:"MA_URL" json:"maUrl"`
	MAImg        string `db:"MA_IMG" json:"maImg"`
	MAStatus     string `db:"OPEN_YN" json:"maStatus"`
	ADTier       string `db:"AD_TIER" json:"adTier"`
	ADTitleLabel string `db:"AD_TITLE_LABEL" json:"adTitleLabel"`
	MAIndx       int    `db:"INDX" json:"maIndx"`
}

type AdminAdInsert struct {
	MAName       string
	MAURL        string
	MAImg        string
	MAStatus     string
	ADTier       string
	ADTitleLabel string
	MAIndx       int
}

type AdminAdStats struct {
	MASeq      int `json:"maSeq"`
	ViewCount  int `json:"viewCount"`
	ClickCount int `json:"clickCount"`
}

// --- Member ---

type AdminMemberRow struct {
	USRSeq    int    `db:"USR_SEQ" json:"usrSeq"`
	USRID     string `db:"USR_ID" json:"usrId"`
	USRName   string `db:"USR_NAME" json:"usrName"`
	USRStatus string `db:"USR_STATUS" json:"usrStatus"`
	USRFN     string `db:"USR_FN" json:"usrFn"`
	USRPhone  string `db:"USR_PHONE" json:"usrPhone"`
	USREmail  string `db:"USR_EMAIL" json:"usrEmail"`
	VisitDate string `db:"VISIT_DATE" json:"visitDate"`
}

type AdminMemberDetail struct {
	USRSeq    int    `db:"USR_SEQ" json:"usrSeq"`
	USRID     string `db:"USR_ID" json:"usrId"`
	USRName   string `db:"USR_NAME" json:"usrName"`
	USRStatus string `db:"USR_STATUS" json:"usrStatus"`
	USRFN     string `db:"USR_FN" json:"usrFn"`
	USRPhone  string `db:"USR_PHONE" json:"usrPhone"`
	USREmail  string `db:"USR_EMAIL" json:"usrEmail"`
	USRNick   string `db:"USR_NICK" json:"usrNick"`
	USRPhoto  string `db:"USR_PHOTO" json:"usrPhoto"`
	RegDate   string `db:"REG_DATE" json:"regDate"`
	VisitCnt  int    `db:"VISIT_CNT" json:"visitCnt"`
	VisitDate string `db:"VISIT_DATE" json:"visitDate"`
}

// --- Dashboard ---

type DashboardStats struct {
	TotalMembers       int              `json:"totalMembers"`
	KakaoLinkedMembers int              `json:"kakaoLinkedMembers"`
	RecentLoginCount   int              `json:"recentLoginCount"`
	PendingApprovals   int              `json:"pendingApprovals"`
	TotalNotices       int              `json:"totalNotices"`
	Donation           DonationSummary  `json:"donation"`
	AdStats            DashboardAdStats `json:"adStats"`
}

type DashboardAdStats struct {
	TotalImpressions int     `json:"totalImpressions"`
	TotalClicks      int     `json:"totalClicks"`
	CTR              float64 `json:"ctr"`
}

// --- Donation Order ---

// AdminDonationOrderRow represents a donation order in the admin list.
type AdminDonationOrderRow struct {
	OSeq    int    `db:"O_SEQ" json:"oSeq"`
	USRSeq  int    `db:"USR_SEQ" json:"usrSeq"`
	USRName string `db:"USR_NAME" json:"usrName"`
	Amount  int    `db:"O_PRICE" json:"amount"`
	PayType string `db:"O_PAY_TYPE" json:"payType"`
	Payment string `db:"O_PAYMENT" json:"payment"`
	PayDate string `db:"PAY_DATE" json:"payDate"`
	Gate    string `db:"O_GATE" json:"gate"`
	RegDate string `db:"REG_DATE" json:"regDate"`
}
