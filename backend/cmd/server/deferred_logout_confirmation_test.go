package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

func confirmedLogoutRequest(t *testing.T, s *goldenServer, proof, device string, wantStatus int, wantResult string) {
	t.Helper()
	body := map[string]string{"refreshToken": proof}
	if device != "" {
		body["deviceToken"] = device
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, s.server.URL+singleProofLogoutPath, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Client clock/expiry opinions must never override the server proof boundary.
	req.Header.Set("X-Client-Test-Clock", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	response, err := s.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus || response.Header.Get("X-Session-Logout-Result") != wantResult {
		t.Fatalf("status=%d result=%q want=%d/%q", response.StatusCode, response.Header.Get("X-Session-Logout-Result"), wantStatus, wantResult)
	}
	if len(response.Cookies()) != 0 {
		t.Fatal("proof logout issued cookies")
	}
	if wantStatus == 204 {
		var extra [1]byte
		if n, _ := response.Body.Read(extra[:]); n != 0 {
			t.Fatal("204 minted response body")
		}
	}
}

func TestDeferredLogoutHTTPConfirmationDistinguishesNoOpFromCommittedRevocation(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	live := pushScopeSession(t, s, 100)
	confirmedLogoutRequest(t, s, live.RefreshToken, "", 204, "confirmed")
	// Repeated retained proof validation must commit too, even without a device.
	confirmedLogoutRequest(t, s, live.RefreshToken, "", 204, "confirmed")
	pushScopeRow(t, s, 100, "legacy-retry", nil)
	confirmedLogoutRequest(t, s, live.RefreshToken, "legacy-retry", 204, "confirmed")
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE DEVICE_TOKEN='legacy-retry'`)
	future := time.Now().Add(time.Hour).Truncate(time.Second)
	missing := deferredSyntheticProof(t, 100, strings.Repeat("d", 32), strings.Repeat("e", 32), future.Unix())
	confirmedLogoutRequest(t, s, missing, "", 204, "unconfirmed")
	// Original expiry ends authority while a rotated successor can still be live.
	originalExpiry := time.Now().Add(-time.Second).Truncate(time.Second)
	sid := strings.Repeat("a", 32)
	jti := strings.Repeat("b", 32)
	next := strings.Repeat("c", 32)
	repo := repository.NewAuthRepository(s.db)
	if err := repo.InsertMobileRefreshToken(100, sid, jti, originalExpiry); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertMobileRefreshToken(100, sid, next, future); err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET CONSUMED_AT=NOW(),ROTATED_TO_JTI=? WHERE MRT_JTI=?`, next, jti)
	proof := deferredSyntheticProof(t, 100, sid, jti, originalExpiry.Unix())
	confirmedLogoutRequest(t, s, proof, "", 204, "unconfirmed")
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? AND REVOKED_AT IS NULL`, next)
	confirmedLogoutRequest(t, s, live.RefreshToken+"x", "", 401, "")
}

func TestDeferredLogoutHTTPConfirmationRequiresSuccessfulCommit(t *testing.T) {
	fault := &atomic.Bool{}
	s := newGoldenServerWithDatabase(t, func(database *mariadb.Database) *sqlx.DB {
		name := fmt.Sprintf("confirmation-fault-%d", deferredFaultDriverSequence.Add(1))
		sql.Register(name, &deferredCommitFaultDriver{fail: fault})
		db, err := sqlx.Open(name, database.DSN)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	})
	pushScopeSeed(t, s)
	live := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, live, "synthetic")
	fault.Store(true)
	confirmedLogoutRequest(t, s, live.RefreshToken, "synthetic", 500, "")
	if fault.Load() {
		t.Fatal("commit fault not exercised")
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? AND REVOKED_AT IS NULL`, live.JTI)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE DEVICE_TOKEN='synthetic'`)
	confirmedLogoutRequest(t, s, live.RefreshToken, "synthetic", 204, "confirmed")
	// Already-revoked validation cannot emit confirmation before its own commit.
	fault.Store(true)
	confirmedLogoutRequest(t, s, live.RefreshToken, "", 500, "")
	if fault.Load() {
		t.Fatal("retained replay bypassed commit")
	}
	confirmedLogoutRequest(t, s, live.RefreshToken, "", 204, "confirmed")
}

func TestDeferredLogoutHTTPConfirmationDoesNotExtendProofExpiredInTransit(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	repo := repository.NewAuthRepository(s.db)
	sid := strings.Repeat("a", 32)
	jti := strings.Repeat("b", 32)
	next := strings.Repeat("c", 32)
	expiry := time.Now().Add(3 * time.Second).Truncate(time.Second)
	if err := repo.InsertMobileRefreshToken(100, sid, jti, expiry); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertMobileRefreshToken(100, sid, next, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET CONSUMED_AT=NOW(),ROTATED_TO_JTI=? WHERE MRT_JTI=?`, next, jti)
	tx, err := s.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var held string
	if err := tx.Get(&held, `SELECT MRT_JTI FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? FOR UPDATE`, jti); err != nil {
		t.Fatal(err)
	}
	proof := deferredSyntheticProof(t, 100, sid, jti, expiry.Unix())
	payload, _ := json.Marshal(map[string]string{"refreshToken": proof})
	type result struct {
		status       int
		confirmation string
		err          error
	}
	done := make(chan result, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodPost, s.server.URL+singleProofLogoutPath, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		response, err := s.client.Do(req)
		if err != nil {
			done <- result{err: err}
			return
		}
		defer response.Body.Close()
		done <- result{status: response.StatusCode, confirmation: response.Header.Get("X-Session-Logout-Result")}
	}()
	select {
	case early := <-done:
		t.Fatalf("request bypassed proof lock: status=%d err=%v", early.status, early.err)
	case <-time.After(100 * time.Millisecond):
	}
	// The client dispatched a valid proof; authority ends while the row is held.
	time.Sleep(time.Until(expiry.Add(100 * time.Millisecond)))
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	select {
	case outcome := <-done:
		if outcome.err != nil || outcome.status != 204 || outcome.confirmation != "unconfirmed" {
			t.Fatalf("expiry outcome=%+v", outcome)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expired proof request stuck")
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? AND REVOKED_AT IS NULL`, next)
}
