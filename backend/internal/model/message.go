// message.go — Alumni direct message domain model
package model

// Message represents a row in ALUMNI_MESSAGE table.
type Message struct {
	AMSeq      int    `db:"AM_SEQ" json:"amSeq"`
	SenderSeq  int    `db:"AM_SENDER_SEQ" json:"senderSeq"`
	RecvrSeq   int    `db:"AM_RECVR_SEQ" json:"recvrSeq"`
	Content    string `db:"AM_CONTENT" json:"content"`
	ReadYN     string `db:"AM_READ_YN" json:"readYn"`
	DelSender  string `db:"AM_DEL_SENDER" json:"-"`
	DelRecvr   string `db:"AM_DEL_RECVR" json:"-"`
	RegDate    string `db:"REG_DATE" json:"regDate"`
	ReadDate   string `db:"READ_DATE" json:"readDate"`
	SenderName string `db:"SENDER_NAME" json:"senderName"`
	RecvrName  string `db:"RECVR_NAME" json:"recvrName"`
}

// SendMessageRequest is the request body for POST /api/messages.
type SendMessageRequest struct {
	RecvrSeq int    `json:"recvrSeq"`
	Content  string `json:"content"`
}

// MessageListResponse is the API response for inbox/outbox listing.
type MessageListResponse struct {
	Items      []Message `json:"items"`
	TotalCount int       `json:"totalCount"`
	Page       int       `json:"page"`
	Size       int       `json:"size"`
	TotalPages int       `json:"totalPages"`
}
