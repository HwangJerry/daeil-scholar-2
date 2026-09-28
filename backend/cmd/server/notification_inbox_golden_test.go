package main

import (
	"net/http"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
)

// TestGoldenNotificationInbox drives the inbox through the real router with push
// disabled (as in the harness config): an operator publishes a notice, every open
// stream gets notification.created, the badge and list show it unread, and
// POST /api/notifications/seen clears it only through the SEQ the app displayed.
func TestGoldenNotificationInbox(t *testing.T) {
	s := newGoldenServer(t)
	operator, member := defaultGoldenMember, goldenRecipient
	seedGoldenMemberAs(t, s.db, operator)
	seedGoldenMemberAs(t, s.db, member)
	goldenExec(t, s.db, `INSERT INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT)
		VALUES (?, 'operator', NOW(), NOW())`, operator.seq)
	_, operatorSession := s.loginAs(t, operator, "android")
	_, memberSession := s.loginAs(t, member, "ios")

	operatorStream := s.openStream(t, operatorSession.AccessToken, "android")
	memberStream := s.openStream(t, memberSession.AccessToken, "ios")
	operatorStream.next(t, "ready")
	memberStream.next(t, "ready")

	body := s.request(t, http.MethodPost, "/api/admin/feed", map[string]any{
		"subject": "합성 새 소식", "contentMd": "합성 본문",
	}, operatorSession.AccessToken, "android", "100", http.StatusCreated)
	noticeSeq := decodeGolden[map[string]int](t, body)["seq"]
	if noticeSeq <= 0 {
		t.Fatalf("notice create must return its seq: %s", body)
	}

	// Push is disabled here, so these frames prove the realtime path stands alone.
	created := memberStream.next(t, "notification.created")
	operatorCreated := operatorStream.next(t, "notification.created")
	want := map[string]any{
		"notificationId": "notice-1", "postSeq": float64(noticeSeq), "type": "admin.notice",
		"title": "새 소식", "body": "합성 새 소식", "deepLink": "/feed/1",
	}
	assertRealtimePayload(t, created, want)
	assertRealtimePayload(t, operatorCreated, want)
	golden.Assert(t, "realtime_notification_created", realtimeGolden(t, created, created.id-1))

	unread := s.request(t, http.MethodGet, "/api/badges", nil, memberSession.AccessToken, "ios", "100", http.StatusOK)
	if badges := decodeGolden[model.BadgeResponse](t, unread); badges.UnreadNotifications != 1 || badges.UnreadMessages != 0 {
		t.Fatalf("badges before seen: %s", unread)
	}
	golden.Assert(t, "badges_unread_notification", unread)

	list := s.request(t, http.MethodGet, "/api/notifications", nil, memberSession.AccessToken, "ios", "100", http.StatusOK)
	inbox := decodeGolden[model.NotificationListResponse](t, list)
	if len(inbox.Items) != 1 || !inbox.Items[0].IsUnread || inbox.Items[0].PostSeq != noticeSeq || inbox.HasMore || inbox.NextCursor != nil {
		t.Fatalf("inbox before seen: %s", list)
	}
	golden.Assert(t, "notifications_list", list)

	invalid := s.request(t, http.MethodGet, "/api/notifications?cursor=page_2", nil, memberSession.AccessToken, "ios", "100", http.StatusBadRequest)
	golden.Assert(t, "notifications_invalid_cursor_400", invalid)

	badSeen := s.request(t, http.MethodPost, "/api/notifications/seen", map[string]any{}, memberSession.AccessToken, "ios", "100", http.StatusBadRequest)
	golden.Assert(t, "notifications_seen_invalid_400", badSeen)

	// A second notice lands while the member's inbox still shows only the first:
	// seen-through the displayed SEQ must leave the new one unread.
	s.request(t, http.MethodPost, "/api/admin/feed", map[string]any{
		"subject": "합성 두 번째 소식", "contentMd": "합성 본문",
	}, operatorSession.AccessToken, "android", "100", http.StatusCreated)
	memberStream.next(t, "notification.created")
	s.request(t, http.MethodPost, "/api/notifications/seen", map[string]any{"lastSeenPostSeq": noticeSeq}, memberSession.AccessToken, "ios", "100", http.StatusNoContent)
	assertInboxBadge(t, s, memberSession.AccessToken, "ios", 1)
	after := decodeGolden[model.NotificationListResponse](t, s.request(t, http.MethodGet, "/api/notifications", nil, memberSession.AccessToken, "ios", "100", http.StatusOK))
	if len(after.Items) != 2 || !after.Items[0].IsUnread || after.Items[1].IsUnread || after.Items[1].PostSeq != noticeSeq {
		t.Fatalf("only the notice published after the displayed one stays unread: %#v", after)
	}

	// A bogus large SEQ is clamped to the newest notice, so it clears the rest
	// but cannot pre-mark future notices.
	s.request(t, http.MethodPost, "/api/notifications/seen", map[string]any{"lastSeenPostSeq": 1 << 30}, memberSession.AccessToken, "ios", "100", http.StatusNoContent)
	assertInboxBadge(t, s, memberSession.AccessToken, "ios", 0)
	s.request(t, http.MethodPost, "/api/admin/feed", map[string]any{
		"subject": "합성 세 번째 소식", "contentMd": "합성 본문",
	}, operatorSession.AccessToken, "android", "100", http.StatusCreated)
	assertInboxBadge(t, s, memberSession.AccessToken, "ios", 1)
	// The operator never reported anything seen, so every notice is unread for them.
	assertInboxBadge(t, s, operatorSession.AccessToken, "android", 3)
}

func assertInboxBadge(t *testing.T, s *goldenServer, accessToken, platform string, want int) {
	t.Helper()
	badges := decodeGolden[model.BadgeResponse](t, s.request(t, http.MethodGet, "/api/badges", nil, accessToken, platform, "100", http.StatusOK))
	if badges.UnreadNotifications != want {
		t.Fatalf("unreadNotifications = %d, want %d", badges.UnreadNotifications, want)
	}
}
