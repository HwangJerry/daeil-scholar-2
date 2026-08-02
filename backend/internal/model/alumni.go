package model

import (
	"database/sql"
)

// AlumniRecord represents a member from WEO_MEMBER.
type AlumniRecord struct {
	USRSeq         int            `db:"USR_SEQ"`
	USRName        string         `db:"USR_NAME"`
	USRPhoto       sql.NullString `db:"USR_PHOTO"`
	GraduationYear sql.NullInt64  `db:"GRADUATION_YEAR"`
	Cohort         sql.NullString `db:"COHORT"`
	Department     sql.NullString `db:"DEPARTMENT"`
	AJCName        sql.NullString `db:"AJC_NAME"`
	USRPosition    sql.NullString `db:"USR_POSITION"`
	USRPhone       sql.NullString `db:"USR_PHONE"`
	USREmail       sql.NullString `db:"USR_EMAIL"`
	USRPhonePublic sql.NullString `db:"USR_PHONE_PUBLIC"`
	USREmailPublic sql.NullString `db:"USR_EMAIL_PUBLIC"`
	BlockedByMe    bool           `db:"BLOCKED_BY_ME"`
}

// AlumniCard is the API response for a single alumni in the search results.
type AlumniCard struct {
	UserSeq     int     `json:"userSeq"`
	Name        string  `json:"name"`
	PhotoURL    *string `json:"photoUrl"`
	Cohort      string  `json:"cohort"`
	Department  string  `json:"department"`
	JobCategory string  `json:"jobCategory"`
	JobRole     string  `json:"jobRole"`
}

// AlumniSearchParams holds the query parameters for alumni search.
type AlumniSearchParams struct {
	Name           string
	GraduationYear int
	Cohort         string
	Department     string
	JobCategory    int
	JobRole        string
	Page           int
	Size           int
}

// AlumniSearchResponse is the API response for GET /api/alumni.
type AlumniSearchResponse struct {
	Items      []AlumniCard `json:"items"`
	Page       int          `json:"page"`
	Size       int          `json:"size"`
	TotalCount int          `json:"totalCount"`
	TotalPages int          `json:"totalPages"`
}

type AlumniDetail struct {
	UserSeq     int              `json:"userSeq"`
	Name        string           `json:"name"`
	PhotoURL    *string          `json:"photoUrl"`
	Cohort      string           `json:"cohort"`
	Department  string           `json:"department"`
	JobCategory string           `json:"jobCategory"`
	JobRole     string           `json:"jobRole"`
	Phone       *string          `json:"phone,omitempty"`
	Email       *string          `json:"email,omitempty"`
	BlockState  AlumniBlockState `json:"blockState"`
}

type AlumniBlockState struct {
	BlockedByMe bool `json:"blockedByMe"`
}

// AlumniFilters holds the available filter options for alumni search.
type AlumniFilters struct {
	GraduationYears []int         `json:"graduationYears"`
	Cohorts         []string      `json:"cohorts"`
	Departments     []string      `json:"departments"`
	JobCategories   []JobCategory `json:"jobCategories"`
	JobRoles        []string      `json:"jobRoles"`
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

// AlumniWidgetItem is a minimal approved-alumni widget entry (name only).
type AlumniWidgetItem struct {
	FmName string `json:"fmName"`
}

// AlumniWidgetResponse is the API response for GET /api/alumni/widget.
type AlumniWidgetResponse struct {
	Items      []AlumniWidgetItem `json:"items"`
	TotalCount int                `json:"totalCount"`
}
