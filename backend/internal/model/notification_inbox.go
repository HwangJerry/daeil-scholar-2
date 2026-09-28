// notification_inbox.go — Models for the in-app notification inbox (GET /api/notifications)
package model

import "time"

// NotificationTypeAdminNotice is the only inbox type: an administrator's 새 소식
// announcement. It matches the push payload type so the apps route both alike.
const NotificationTypeAdminNotice = "admin.notice"

// NoticeNotificationRow is one published notice read for the inbox.
type NoticeNotificationRow struct {
	SEQ     int       `db:"SEQ"`
	Subject string    `db:"SUBJECT"`
	RegDate time.Time `db:"REG_DATE"`
}

// NotificationItem is one inbox entry, shaped like the push the member received.
type NotificationItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	IsUnread  bool   `json:"isUnread"`
	DeepLink  string `json:"deepLink"`
	PostSeq   int    `json:"postSeq"`
}

// NotificationListResponse is the API response for GET /api/notifications.
type NotificationListResponse struct {
	Items      []NotificationItem `json:"items"`
	NextCursor *string            `json:"nextCursor"`
	HasMore    bool               `json:"hasMore"`
}
