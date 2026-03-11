// comment_notifier.go — Notification dispatch for comment events on posts
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

// CommentNotifier handles sending notifications when comment events occur.
type CommentNotifier struct {
	feedRepo *repository.FeedRepository
	notifSvc *NotificationService
}

// NewCommentNotifier creates a CommentNotifier.
func NewCommentNotifier(feedRepo *repository.FeedRepository, notifSvc *NotificationService) *CommentNotifier {
	return &CommentNotifier{feedRepo: feedRepo, notifSvc: notifSvc}
}

// OnCommentCreated notifies the post owner (in a goroutine) when a comment is created.
// Skips notification if the commenter is the post owner.
func (n *CommentNotifier) OnCommentCreated(postSeq, commenterSeq int, commenterName string) {
	if n.notifSvc == nil {
		return
	}
	go n.notifyPostOwner(postSeq, commenterSeq, commenterName, model.NotiTypeNewComment)
}

// notifyPostOwner looks up the post author and sends a notification if different from the actor.
func (n *CommentNotifier) notifyPostOwner(postSeq, actorSeq int, actorName, notiType string) {
	ownerSeq, err := n.feedRepo.GetPostOwnerSeq(postSeq)
	if err != nil || ownerSeq == 0 || ownerSeq == actorSeq {
		return
	}
	if notiType == model.NotiTypeNewComment {
		n.notifSvc.NotifyNewComment(ownerSeq, actorName, postSeq)
	}
}
