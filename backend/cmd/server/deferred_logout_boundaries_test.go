package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func deferredSyntheticProof(t *testing.T, account int, sid, jti string, expiry int64) string {
	t.Helper()
	now := time.Now().Add(-time.Hour).Unix()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iss": "dflh-saf-v2-backend", "aud": "dflh-saf-v2-mobile", "sub": accountString(account), "typ": "refresh", "ver": 1, "iat": now, "nbf": now, "exp": expiry, "sid": sid, "jti": jti}).SignedString([]byte("ts07-synthetic-jwt-secret-never-used-outside-tests"))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestDeferredLogoutHTTPExpiryBindingTamperAndRateLimit(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := repository.NewAuthRepository(s.db)
	user, err := repo.GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour).Unix()
	cases := []struct {
		name, token string
		status      int
	}{
		{"wrong-account", deferredSyntheticProof(t, goldenMemberID+1, session.SID, session.JTI, session.RefreshExpiresAt), 401},
		{"wrong-sid", deferredSyntheticProof(t, goldenMemberID, strings.Repeat("f", 32), session.JTI, session.RefreshExpiresAt), 401},
		{"wrong-original-expiry", deferredSyntheticProof(t, goldenMemberID, session.SID, session.JTI, future), 401},
		{"tamper", session.RefreshToken + "A", 401},
		{"wrong-type", session.AccessToken, 401},
		{"missing", deferredSyntheticProof(t, goldenMemberID, strings.Repeat("f", 32), strings.Repeat("e", 32), future), 204},
		{"expired-original-with-active-successor", deferredSyntheticProof(t, goldenMemberID, session.SID, session.JTI, time.Now().Add(-time.Minute).Unix()), 204},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": tc.token}, "", "ios", "100", tc.status)
		})
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL`, goldenMemberID)
	// The expired proof no-op above cannot extend authorization to a live family.
	// The route's limiter is distinct from refresh/login, and bounds even valid replays.
	for n := len(cases); n < 10; n++ {
		s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": "invalid"}, "", "ios", "100", 401)
	}
	s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 429)
	s.request(t, http.MethodPost, "/api/auth/refresh", map[string]string{"refreshToken": "invalid"}, "", "ios", "100", 401)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, goldenMemberID)
}

func accountString(account int) string { return fmt.Sprintf("%d", account) }

func TestDeferredLogoutHTTPConcurrentSuccessorRotationAndReplay(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	issuer := service.NewMobileSessionIssuer(s.deps.authService)
	unrelated, err := issuer.Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		original, err := issuer.Issue(user)
		if err != nil {
			t.Fatal(err)
		}
		successor, err := issuer.Rotate(original.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		start := make(chan struct{})
		statuses := make(chan int, 1)
		rotateErrors := make(chan error, 1)
		go func() { defer wg.Done(); <-start; _, err := issuer.Rotate(successor.RefreshToken); rotateErrors <- err }()
		go func() {
			defer wg.Done()
			<-start
			payload, _ := json.Marshal(map[string]string{"refreshToken": original.RefreshToken})
			req, _ := http.NewRequest(http.MethodPost, s.server.URL+"/api/auth/logout/deferred", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := s.client.Do(req)
			if err != nil {
				statuses <- 0
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if len(resp.Cookies()) != 0 {
				statuses <- 0
				return
			}
			statuses <- resp.StatusCode
		}()
		close(start)
		wg.Wait()
		status := <-statuses
		rotateErr := <-rotateErrors
		if rotateErr != nil && !errors.Is(rotateErr, repository.ErrRefreshTokenReplay) {
			var conflict *mysql.MySQLError
			if !errors.As(rotateErr, &conflict) || conflict.Number != 1213 {
				t.Fatalf("unexpected rotation failure: %v", rotateErr)
			}
		}
		if status != 204 && status != 500 {
			t.Fatalf("concurrent status=%d", status)
		}
		// A deadlock/transient DB failure may require retry; it cannot claim success.
		s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": original.RefreshToken}, "", "ios", "100", 204)
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL`, goldenMemberID, original.SID)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND MRT_SID=? AND REVOKED_AT IS NULL`, goldenMemberID, unrelated.SID)
	s.request(t, http.MethodGet, "/api/auth/me", nil, unrelated.AccessToken, "ios", "100", 200)
}

func TestDeferredLogoutHTTPExpiredAncestorDoesNotRevokeExtendedFamily(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := repository.NewAuthRepository(s.db)
	sid, jti, next := strings.Repeat("c", 32), strings.Repeat("d", 32), strings.Repeat("e", 32)
	expired := time.Now().Add(-time.Minute).Truncate(time.Second)
	later := time.Now().Add(time.Hour).Truncate(time.Second)
	if err := repo.InsertMobileRefreshToken(goldenMemberID, sid, jti, expired); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertMobileRefreshToken(goldenMemberID, sid, next, later); err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET CONSUMED_AT=NOW(),ROTATED_TO_JTI=? WHERE MRT_JTI=?`, next, jti)
	proof := deferredSyntheticProof(t, goldenMemberID, sid, jti, expired.Unix())
	s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": proof}, "", "ios", "100", 204)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND MRT_SID=? AND EXPIRES_AT>NOW() AND REVOKED_AT IS NULL`, goldenMemberID, sid)
	goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, goldenMemberID)
}

func TestDeferredLogoutHTTPBlockedBuildCanEndSession(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	seedGoldenPolicy(t, s, "ios")
	user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	s.request(t, http.MethodGet, "/api/auth/me", nil, session.AccessToken, "ios", "99", 426)
	s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "99", 204)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL`, goldenMemberID)
}
