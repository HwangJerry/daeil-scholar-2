package model

import (
	"database/sql"
)

// AlumniRecord represents a member from WEO_MEMBER.
type AlumniRecord struct {
	USRSeq        sql.NullInt64  `db:"USR_SEQ"`
	USRName       string         `db:"USR_NAME"`
	USRFN         sql.NullString `db:"USR_FN"`
	USRDept       sql.NullString `db:"USR_DEPT"`
	USRBizName    sql.NullString `db:"USR_BIZ_NAME"`
	USRBizDesc    sql.NullString `db:"USR_BIZ_DESC"`
	USRBizAddr    sql.NullString `db:"USR_BIZ_ADDR"`
	USRPhone      sql.NullString `db:"USR_PHONE"`
	USREmail      sql.NullString `db:"USR_EMAIL"`
	USRPhoto      sql.NullString `db:"USR_PHOTO"`
	USRNick       sql.NullString `db:"USR_NICK"`
	USRPosition   sql.NullString `db:"USR_POSITION"`
	USRBizCard    sql.NullString `db:"USR_BIZ_CARD"`
	USRPhonePublic sql.NullString `db:"USR_PHONE_PUBLIC"`
	USREmailPublic sql.NullString `db:"USR_EMAIL_PUBLIC"`
	AJCName       sql.NullString `db:"AJC_NAME"`
	Tags          sql.NullString `db:"TAGS"`
}

// AlumniCard is the API response for a single alumni in the search results.
type AlumniCard struct {
	FMSeq       int      `json:"fmSeq"`
	FMName      string   `json:"fmName"`
	FMFN        string   `json:"fmFn"`
	FMDept      string   `json:"fmDept"`
	Company     string   `json:"company"`
	Position    string   `json:"position"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	BizName     string   `json:"bizName"`
	BizDesc     string   `json:"bizDesc"`
	BizAddr     string   `json:"bizAddr"`
	JobCatName  string   `json:"jobCatName"`
	Tags        []string `json:"tags"`
	Photo       string   `json:"photo"`
	UsrSeq      int      `json:"usrSeq"`
	Nick        string   `json:"nick"`
	BizCard     string   `json:"bizCard"`
}

// AlumniSearchParams holds the query parameters for alumni search.
type AlumniSearchParams struct {
	FN      string
	Dept    string
	Name    string
	Company string
	JobCat  int
	Page    int
	Size    int
}

// AlumniSearchResponse is the API response for GET /api/alumni.
type AlumniSearchResponse struct {
	Items       []AlumniCard `json:"items"`
	TotalCount  int          `json:"totalCount"`
	WeeklyCount int          `json:"weeklyCount"`
	Page        int          `json:"page"`
	Size        int          `json:"size"`
	TotalPages  int          `json:"totalPages"`
}

// AlumniFilters holds the available filter options for alumni search.
type AlumniFilters struct {
	FNList        []string      `json:"fnList"`
	DeptList      []string      `json:"deptList"`
	JobCategories []JobCategory `json:"jobCategories"`
}

// JobCategory represents a row in ALUMNI_JOB_CATEGORY table.
type JobCategory struct {
	Seq  int    `db:"AJC_SEQ" json:"seq"`
	Name string `db:"AJC_NAME" json:"name"`
}

// UserTag represents a row in ALUMNI_USER_TAG table.
type UserTag struct {
	Seq    int    `db:"AUT_SEQ" json:"seq"`
	USRSeq int    `db:"USR_SEQ" json:"usrSeq"`
	Tag    string `db:"AUT_TAG" json:"tag"`
	Indx   int    `db:"AUT_INDX" json:"indx"`
}

// AlumniWidgetItem is a minimal alumni entry for the public widget (name only).
type AlumniWidgetItem struct {
	FmName string `json:"fmName"`
}

// AlumniWidgetResponse is the API response for GET /api/alumni/widget.
type AlumniWidgetResponse struct {
	Items      []AlumniWidgetItem `json:"items"`
	TotalCount int                `json:"totalCount"`
}
