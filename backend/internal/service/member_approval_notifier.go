// member_approval_notifier.go — Notification dispatch for member registration approval events
package service

// MemberApprovalNotifier handles sending notifications when a member is approved.
type MemberApprovalNotifier struct {
	notifSvc *NotificationService
}

// NewMemberApprovalNotifier creates a MemberApprovalNotifier.
func NewMemberApprovalNotifier(notifSvc *NotificationService) *MemberApprovalNotifier {
	return &MemberApprovalNotifier{notifSvc: notifSvc}
}

// OnApproved triggers an in-app notification and email when a member transitions
// from pending (BBB) to approved (CCC). Runs in a background goroutine.
func (n *MemberApprovalNotifier) OnApproved(usrSeq int, email, name string) {
	if n.notifSvc == nil {
		return
	}
	go n.notifSvc.NotifyRegistrationApproved(usrSeq, email, name)
}
