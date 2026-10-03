package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/golden"
)

// TestGoldenFeedOfficialProfile drives the official-profile byline through the
// real router on MariaDB 10.1: a new post defaults to the foundation's name,
// an explicit false keeps the operator's name, an edit can switch either way
// (turning it off restores the author's member name), an edit without the
// field keeps the byline, and USR_SEQ always stays the real operator.
func TestGoldenFeedOfficialProfile(t *testing.T) {
	s := newGoldenServer(t)
	operator := defaultGoldenMember
	seedGoldenMemberAs(t, s.db, operator)
	goldenExec(t, s.db, `INSERT INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT)
		VALUES (?, 'operator', NOW(), NOW())`, operator.seq)
	_, session := s.loginAs(t, operator, "android")
	token := session.AccessToken

	officialSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 공식 공지", "contentMd": "본문"})
	personalSeq := s.createGoldenNotice(t, token, map[string]any{"subject": "합성 개인 공지", "contentMd": "본문", "officialProfile": false})

	readDetail := func(t *testing.T, seq int) (model.NoticeDetail, []byte) {
		t.Helper()
		body := s.request(t, http.MethodGet, fmt.Sprintf("/api/feed/%d", seq), nil, "", "ios", "100", http.StatusOK)
		return decodeGolden[model.NoticeDetail](t, body), body
	}
	assertByline := func(t *testing.T, seq int, wantOfficial bool, wantName string) {
		t.Helper()
		detail, body := readDetail(t, seq)
		if detail.OfficialProfile != wantOfficial || detail.RegName != wantName {
			t.Fatalf("post %d byline: %s", seq, body)
		}
		var usrSeq int
		if err := s.db.Get(&usrSeq, `SELECT USR_SEQ FROM WEO_BOARDBBS WHERE SEQ = ?`, seq); err != nil || usrSeq != operator.seq {
			t.Fatalf("post %d USR_SEQ = %d, %v; the real operator must stay recorded", seq, usrSeq, err)
		}
	}
	edit := func(t *testing.T, seq int, body map[string]any) {
		t.Helper()
		s.request(t, http.MethodPut, fmt.Sprintf("/api/admin/feed/%d", seq), body, token, "android", "100", http.StatusNoContent)
	}

	t.Run("create", func(t *testing.T) {
		assertByline(t, officialSeq, true, "대일외고장학회")
		assertByline(t, personalSeq, false, operator.name)
		_, body := readDetail(t, officialSeq)
		golden.Assert(t, "feed_detail_official_profile", body)

		body = s.request(t, http.MethodGet, "/api/feed", nil, "", "ios", "100", http.StatusOK)
		feed := decodeGolden[model.FeedResponse](t, body)
		if len(feed.Items) != 2 || feed.Items[0].NoticeItem == nil || feed.Items[0].SEQ != personalSeq ||
			feed.Items[0].OfficialProfile || !feed.Items[1].OfficialProfile {
			t.Fatalf("feed list must carry officialProfile per post: %s", body)
		}
		hero := decodeGolden[model.NoticeItem](t, s.request(t, http.MethodGet, "/api/feed/hero", nil, "", "ios", "100", http.StatusOK))
		if hero.SEQ != personalSeq || hero.OfficialProfile {
			t.Fatalf("hero byline = %+v", hero)
		}
	})
	t.Run("update", func(t *testing.T) {
		edit(t, officialSeq, map[string]any{"subject": "합성 공식 공지", "contentMd": "수정", "officialProfile": false})
		assertByline(t, officialSeq, false, operator.name)

		edit(t, officialSeq, map[string]any{"subject": "합성 공식 공지", "contentMd": "수정"})
		assertByline(t, officialSeq, false, operator.name)

		edit(t, personalSeq, map[string]any{"subject": "합성 개인 공지", "contentMd": "수정", "officialProfile": true})
		assertByline(t, personalSeq, true, "대일외고장학회")
	})
	t.Run("admin_reads", func(t *testing.T) {
		detail := decodeGolden[model.NoticeDetail](t, s.request(t, http.MethodGet, fmt.Sprintf("/api/admin/feed/%d", personalSeq), nil, token, "android", "100", http.StatusOK))
		if !detail.OfficialProfile || detail.RegName != "대일외고장학회" {
			t.Fatalf("admin detail = %+v", detail)
		}
		list := decodeGolden[struct {
			Items []model.AdminNoticeRow `json:"items"`
		}](t, s.request(t, http.MethodGet, "/api/admin/feed", nil, token, "android", "100", http.StatusOK))
		if len(list.Items) != 2 || !list.Items[0].OfficialProfile || list.Items[1].OfficialProfile {
			t.Fatalf("admin list = %+v", list.Items)
		}
	})
}
