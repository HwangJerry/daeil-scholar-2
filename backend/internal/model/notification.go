// notification.go — Notification domain models, response types, and type constants
package model

// Notification represents a row in ALUMNI_NOTIFICATION table.
type Notification struct {
	ANSeq    int    `db:"AN_SEQ" json:"anSeq"`
	USRSeq   int    `db:"USR_SEQ" json:"usrSeq"`
	ANType   string `db:"AN_TYPE" json:"anType"`
	ANTitle  string `db:"AN_TITLE" json:"anTitle"`
	ANBody   string `db:"AN_BODY" json:"anBody"`
	ANRefSeq *int   `db:"AN_REF_SEQ" json:"anRefSeq"`
	ReadYN   string `db:"AN_READ_YN" json:"readYn"`
	RegDate  string `db:"REG_DATE" json:"regDate"`
}

// NotificationListResponse is the paginated response for notification list API.
type NotificationListResponse struct {
	Items       []Notification `json:"items"`
	TotalCount  int            `json:"totalCount"`
	UnreadCount int            `json:"unreadCount"`
	Page        int            `json:"page"`
	Size        int            `json:"size"`
	TotalPages  int            `json:"totalPages"`
}

// BadgeResponse aggregates unread counts for the badge API endpoint.
type BadgeResponse struct {
	UnreadMessages      int `json:"unreadMessages"`
	UnreadNotifications int `json:"unreadNotifications"`
}

// Notification type constants used across the notification system.
const (
	NotiTypeNewMessage           = "NEW_MESSAGE"
	NotiTypeRegistrationApproved = "REGISTRATION_APPROVED"
	NotiTypeNewComment           = "NEW_COMMENT"
	NotiTypeNewLike              = "NEW_LIKE"
)
