package model

import (
	"database/sql"
)

// AlumniRecord represents a member from FUNDAMENTAL_MEMBER or WEO_MEMBER (UNION ALL result).
type AlumniRecord struct {
	FMSEQ      sql.NullInt64  `db:"FM_SEQ" json:"fmSeq"`
	FMName     string         `db:"FM_NAME" json:"fmName"`
	FMFN       sql.NullString `db:"FM_FN" json:"fmFn"`
	FMDept     sql.NullString `db:"FM_DEPT" json:"fmDept"`
	FMCompany  sql.NullString `db:"FM_COMPANY" json:"fmCompany"`
	FMPos      sql.NullString `db:"FM_POSITION" json:"fmPosition"`
	FMPhone    sql.NullString `db:"FM_PHONE" json:"-"`
	FMEmail    sql.NullString `db:"FM_EMAIL" json:"-"`
	FMSMS      sql.NullString `db:"FM_SMS" json:"-"`
	FMSpam     sql.NullString `db:"FM_SPAM" json:"-"`
	USRBizName sql.NullString `db:"USR_BIZ_NAME" json:"-"`
	USRBizDesc sql.NullString `db:"USR_BIZ_DESC" json:"-"`
	USRBizAddr sql.NullString `db:"USR_BIZ_ADDR" json:"-"`
	AJCName    sql.NullString `db:"AJC_NAME" json:"-"`
	AJCColor   sql.NullString `db:"AJC_COLOR" json:"-"`
	USRPhoto   sql.NullString `db:"USR_PHOTO" json:"-"`
	USRSeq     sql.NullInt64  `db:"USR_SEQ" json:"-"`
	Tags       sql.NullString `db:"TAGS" json:"-"`
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
	JobCatColor string   `json:"jobCatColor"`
	Tags        []string `json:"tags"`
	Photo       string   `json:"photo"`
	UsrSeq      int      `json:"usrSeq"`
}

// AlumniSearchParams holds the query parameters for alumni search.
type AlumniSearchParams struct {
	FN       string
	Dept     string
	Name     string
	Company  string
	Position string
	JobCat   int
	Page     int
	Size     int
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
	Seq   int    `db:"AJC_SEQ" json:"seq"`
	Name  string `db:"AJC_NAME" json:"name"`
	Color string `db:"AJC_COLOR" json:"color"`
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
