// interfaces.go — Repository interfaces for dependency injection and testability
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// LikeQuerier defines the methods used by LikeService for like operations.
type LikeQuerier interface {
	HasUserLiked(bbsSeq int, usrSeq int) (bool, error)
	HasAnyLikeRow(bbsSeq int, usrSeq int) (bool, error)
	InsertLike(bbsSeq int, usrSeq int) error
	SetLikeOpenByUser(bbsSeq int, usrSeq int, openYN string) error
}

// FeedQuerier defines the methods used by FeedService for feed operations.
type FeedQuerier interface {
	GetNotices(cursor int, size int, heroSeq int, userSeq int) ([]model.NoticeItem, error)
	GetHeroNotice() (*model.NoticeItem, error)
	GetNoticeDetail(seq int) (*model.NoticeDetail, error)
	IncrementHit(seq int) error
	GetLikeCount(seq int) (int, error)
	GetCommentCount(seq int) (int, error)
	GetPrevPost(seq int) (*model.PostSibling, error)
	GetNextPost(seq int) (*model.PostSibling, error)
	GetFilesByPost(seq int) ([]model.FileRecord, error)
	GetPostOwnerSeq(seq int) (int, error)
}

// MessageQuerier defines the methods used by MessageService for messaging operations.
type MessageQuerier interface {
	InsertMessage(senderSeq int, recvrSeq int, content string) error
	GetInbox(usrSeq int, page int, size int) ([]model.Message, int, error)
	GetOutbox(usrSeq int, page int, size int) ([]model.Message, int, error)
	MarkAsRead(amSeq int, usrSeq int) error
	DeleteMessage(amSeq int, usrSeq int) error
	GetUnreadCount(usrSeq int) (int, error)
}

// ProfileQuerier defines the profile methods used by MessageService for user existence checks.
type ProfileQuerier interface {
	CheckUserExists(usrSeq int) (bool, error)
}

// NotificationQuerier defines the methods used by NotificationService for notification operations.
type NotificationQuerier interface {
	Insert(usrSeq int, notiType, title, body string, refSeq *int) error
	GetByUser(usrSeq, page, size int) ([]model.Notification, int, error)
	GetUnreadCount(usrSeq int) (int, error)
	MarkAsRead(anSeq, usrSeq int) error
	MarkAllAsRead(usrSeq int) error
}

// AuthQuerier defines the auth repository methods used by NotificationService.
type AuthQuerier interface {
	GetMemberBySeq(usrSeq int) (*model.User, error)
}

// PasswordResetQuerier defines the methods used by PasswordResetService for password reset operations.
type PasswordResetQuerier interface {
	FindMemberByEmail(email string) (*model.User, error)
	InsertToken(usrSeq int, token string, expiresAt time.Time) error
	FindValidToken(token string) (*model.PasswordResetToken, error)
	MarkTokenUsed(token string) error
	UpdatePassword(usrSeq int, hashedPwd string) error
	GetMemberNameBySeq(usrSeq int) (string, error)
}
