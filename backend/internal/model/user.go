package model

import (
	"database/sql"
	"time"
)

// User represents a member from WEO_MEMBER table.
type User struct {
	USRSeq    int            `db:"USR_SEQ" json:"usrSeq"`
	USRID     string         `db:"USR_ID" json:"usrId"`
	USRName   string         `db:"USR_NAME" json:"usrName"`
	USRStatus string         `db:"USR_STATUS" json:"usrStatus"`
	USRPhone  sql.NullString `db:"USR_PHONE" json:"-"`
	USRFN     sql.NullString `db:"USR_FN" json:"usrFn,omitempty"`
	USREmail  sql.NullString `db:"USR_EMAIL" json:"usrEmail,omitempty"`
	USRNick   sql.NullString `db:"USR_NICK" json:"usrNick,omitempty"`
	USRPhoto  sql.NullString `db:"USR_PHOTO" json:"usrPhoto,omitempty"`
	RegDate   sql.NullTime   `db:"REG_DATE" json:"regDate,omitempty"`
	VisitCnt  int            `db:"VISIT_CNT" json:"-"`
	VisitDate sql.NullTime   `db:"VISIT_DATE" json:"-"`
}

// UserProfile is the public-facing profile for the current user (MyPage).
type UserProfile struct {
	USRSeq         int      `json:"usrSeq"`
	USRName        string   `json:"usrName"`
	USRNick        string   `json:"usrNick"`
	USRPhone       string   `json:"usrPhone"`
	USREmail       string   `json:"usrEmail"`
	USRFN          string   `json:"usrFn"`
	USRPhoto       string   `json:"usrPhoto"`
	BizName        string   `json:"bizName"`
	BizDesc        string   `json:"bizDesc"`
	BizAddr        string   `json:"bizAddr"`
	Position       string   `json:"position"`
	JobCat         int      `json:"jobCat"`
	JobCatName     string   `json:"jobCatName"`
	Tags           []string `json:"tags"`
	FmDept         string   `json:"fmDept"`         // WEO_MEMBER.USR_DEPT (예: "영어과")
	RegDate        string   `json:"regDate"`        // 가입일 포맷 "YYYY. MM" (예: "2024. 03")
	USRPhonePublic string   `json:"usrPhonePublic"` // 'Y' | 'N'
	USREmailPublic string   `json:"usrEmailPublic"` // 'Y' | 'N'
	USRBizCard     string   `json:"usrBizCard"`     // 명함 이미지 URL
	HasPassword    bool     `json:"hasPassword"`    // false for Kakao-only users (USR_PWD = '')
	HasSocialLogin bool     `json:"hasSocialLogin"` // true if WEO_MEMBER_SOCIAL record exists
}

// ProfileUpdateRequest is the request body for PUT /api/profile.
type ProfileUpdateRequest struct {
	USRName        string   `json:"usrName"`
	USRFN          string   `json:"usrFn"`
	USRPhone       string   `json:"usrPhone"`
	USREmail       string   `json:"usrEmail"`
	BizName        string   `json:"bizName"`
	BizDesc        string   `json:"bizDesc"`
	BizAddr        string   `json:"bizAddr"`
	Position       string   `json:"position"`
	FmDept         string   `json:"fmDept"`
	JobCat         *int     `json:"jobCat"`
	Tags           []string `json:"tags"`
	USRPhonePublic string   `json:"usrPhonePublic"` // 'Y' | 'N'
	USREmailPublic string   `json:"usrEmailPublic"` // 'Y' | 'N'
}

// AuthUser is the minimal user info stored in context after authentication.
type AuthUser struct {
	USRSeq    int    `json:"usrSeq"`
	USRID     string `json:"usrId"`
	USRName   string `json:"usrName"`
	USRStatus string `json:"usrStatus"`
}

// JWTClaims represents the JWT token claims.
type JWTClaims struct {
	USRSeq    int    `json:"usrSeq"`
	USRName   string `json:"usrName"`
	USRStatus string `json:"usrStatus"`
}

// UserSession represents a row in USER_SESSION table.
type UserSession struct {
	SessionID string    `db:"SESSION_ID"`
	USRSeq    int       `db:"USR_SEQ"`
	Provider  string    `db:"PROVIDER"`
	ExpiresAt time.Time `db:"EXPIRES_AT"`
	CreatedAt time.Time `db:"CREATED_AT"`
}

// MemberSocial represents a row in WEO_MEMBER_SOCIAL table.
type MemberSocial struct {
	NMSSeq  int            `db:"NMS_SEQ"`
	USRSeq  int            `db:"USR_SEQ"`
	NMSGate string         `db:"NMS_GATE"`
	NMSID   string         `db:"NMS_ID"`
	RegDate sql.NullTime   `db:"REG_DATE"`
	NMSName sql.NullString `db:"NMS_NAME"`
}
