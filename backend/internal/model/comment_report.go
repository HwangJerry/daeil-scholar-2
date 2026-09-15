// comment_report.go — Abuse report requests and moderator queue records.
package model

type CommentReportRequest struct {
	PostID    int64  `json:"-"`
	CommentID int64  `json:"-"`
	Reason    string `json:"reason"`
	Details   string `json:"details"`
}

type CommentReportResolution struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type CommentReport struct {
	ID            int64  `db:"REPORT_ID" json:"id"`
	PostID        int64  `db:"POST_SEQ" json:"postId"`
	CommentID     int64  `db:"COMMENT_SEQ" json:"commentId"`
	Visible       bool   `db:"VISIBLE" json:"visible"`
	ReporterSeq   int    `db:"REPORTER_SEQ" json:"reporterSeq"`
	ReportedSeq   int    `db:"REPORTED_SEQ" json:"reportedSeq"`
	Reason        string `db:"REASON" json:"reason"`
	Details       string `db:"DETAILS" json:"details"`
	Content       string `db:"CONTENT_SNAPSHOT" json:"content"`
	Status        string `db:"STATUS" json:"status"`
	ModeratorNote string `db:"MODERATOR_NOTE" json:"moderatorNote"`
	CreatedAt     string `db:"CREATED_AT" json:"createdAt"`
}
