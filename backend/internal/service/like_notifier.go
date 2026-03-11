// like_notifier.go — Notification dispatch for like events on posts
package service

import "github.com/dflh-saf/backend/internal/repository"

// LikeNotifier handles sending notifications when like events occur.
type LikeNotifier struct {
	feedRepo repository.FeedQuerier
	notifSvc *NotificationService
}

// NewLikeNotifier creates a LikeNotifier.
func NewLikeNotifier(feedRepo repository.FeedQuerier, notifSvc *NotificationService) *LikeNotifier {
	return &LikeNotifier{feedRepo: feedRepo, notifSvc: notifSvc}
}

// OnLiked notifies the post owner (in a goroutine) when a post is liked.
// Skips notification if the liker is the post owner.
func (n *LikeNotifier) OnLiked(postSeq, likerSeq int, likerName string) {
	if n.notifSvc == nil {
		return
	}
	go func() {
		ownerSeq, err := n.feedRepo.GetPostOwnerSeq(postSeq)
		if err != nil || ownerSeq == 0 || ownerSeq == likerSeq {
			return
		}
		n.notifSvc.NotifyNewLike(ownerSeq, likerName, postSeq)
	}()
}
