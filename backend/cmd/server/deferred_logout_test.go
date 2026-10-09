package main

import (
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"testing"
	"time"
)

func TestDeferredLogoutHTTPRevokesConsumedAncestorOnly(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := repository.NewAuthRepository(s.db)
	user, err := repo.GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	issuer := service.NewMobileSessionIssuer(s.deps.authService)
	original, err := issuer.Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	successor, err := issuer.Rotate(original.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	unrelated, err := issuer.Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	seedGoldenMemberAs(t, s.db, goldenMemberSeed{seq: 101, usrID: "unrelatedSynthetic", name: "Other Synthetic", phone: "01000000003", email: "other@example.test"})
	if err := repo.InsertMobileRefreshToken(101, original.SID, "ffffffffffffffffffffffffffffffff", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	before := 3
	goldenCount(t, s.db, before, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, goldenMemberID)
	for range 2 {
		body := s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": original.RefreshToken}, "", "ios", "100", 204)
		if len(body) != 0 {
			t.Fatal("revoke minted body")
		}
	}
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL`, goldenMemberID, original.SID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL`, goldenMemberID, unrelated.SID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=101 AND MRT_SID=? AND REVOKED_AT IS NULL`, original.SID)
	goldenCount(t, s.db, before, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, goldenMemberID)
	s.request(t, http.MethodGet, "/api/auth/me", nil, successor.AccessToken, "ios", "100", 401)
	s.request(t, http.MethodGet, "/api/auth/me", nil, unrelated.AccessToken, "ios", "100", 200)
	body := s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": successor.RefreshToken}, "", "ios", "100", 401)
	assertGoldenError(t, body, "REFRESH_REPLAY_DETECTED")
}
