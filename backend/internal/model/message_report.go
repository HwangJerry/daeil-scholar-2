// message_report.go — Abuse report requests and moderator queue records.
package model

type MessageReportRequest struct {
	MessageID int64  `json:"messageId"`
	Reason    string `json:"reason"`
	Details   string `json:"details"`
}

type MessageReportResolution struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type MessageReport struct {
	ID            int64  `db:"REPORT_ID" json:"id"`
	MessageID     int64  `db:"MESSAGE_ID" json:"messageId"`
	ReporterSeq   int    `db:"REPORTER_SEQ" json:"reporterSeq"`
	ReportedSeq   int    `db:"REPORTED_SEQ" json:"reportedSeq"`
	Reason        string `db:"REASON" json:"reason"`
	Details       string `db:"DETAILS" json:"details"`
	Content       string `db:"CONTENT_SNAPSHOT" json:"content"`
	Status        string `db:"STATUS" json:"status"`
	ModeratorNote string `db:"MODERATOR_NOTE" json:"moderatorNote"`
	CreatedAt     string `db:"CREATED_AT" json:"createdAt"`
}
