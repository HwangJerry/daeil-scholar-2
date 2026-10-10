package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

const singleProofLogoutPath = "/api/auth/logout/deferred"
const allProofLogoutPath = "/api/auth/logout/all/deferred"

func pushScopeSession(t *testing.T, s *goldenServer, account int) *model.MobileSession {
	t.Helper()
	user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(account)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	return session
}
func pushScopeSeed(t *testing.T, s *goldenServer) {
	t.Helper()
	seedGoldenMember(t, s.db)
	seedGoldenMemberAs(t, s.db, goldenMemberSeed{seq: 101, usrID: "pushOther", name: "Other Synthetic", phone: "01000000003", email: "other@example.test"})
}
func pushScopeRegister(t *testing.T, s *goldenServer, session *model.MobileSession, token string) {
	t.Helper()
	// This is the unchanged legacy Android request shape; ownership is server-assigned.
	s.request(t, http.MethodPost, "/api/push/device/register", map[string]string{"platform": "android", "deviceToken": token, "locale": "ko-KR"}, session.AccessToken, "android", "100", 200)
}
func pushScopeRow(t *testing.T, s *goldenServer, account int, token string, sid any) {
	t.Helper()
	goldenExec(t, s.db, `INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN (USR_SEQ,PLATFORM,DEVICE_TOKEN,LOCALE,SESSION_SID) VALUES (?,'android',?,'ko-KR',?)`, account, token, sid)
}
func pushScopeLegacyLog(t *testing.T, s *goldenServer, account int) {
	t.Helper()
	goldenExec(t, s.db, `INSERT INTO WEO_MEMBER_LOG (USR_SEQ,LOG_DATE,REG_DATE,REG_GEOCODE,REG_IPADDR,SESSIONID) VALUES (?,NOW(),NOW(),'synthetic','127.0.0.1','synthetic')`, account)
}

func TestPushScopeHTTPOriginalProofCannotDeleteNewOwner(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	old := pushScopeSession(t, s, 100)
	newSession := pushScopeSession(t, s, 100)
	other := pushScopeSession(t, s, 101)
	pushScopeRegister(t, s, old, "shared-device")
	pushScopeRegister(t, s, newSession, "shared-device")
	pushScopeRegister(t, s, newSession, "other-device")
	pushScopeRegister(t, s, other, "other-account")
	pushScopeRow(t, s, 100, "legacy-device", nil)
	// A legacy auth request cannot demote a newly bound row, even from another account.
	repo := repository.NewPushRepository(s.db)
	if err := repo.RegisterDevice(101, model.PushDeviceRegistration{Platform: "android", DeviceToken: "shared-device", Locale: "en"}); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100 AND DEVICE_TOKEN='shared-device' AND SESSION_SID=?`, newSession.SID)
	// An old authenticated unregister that arrived after new registration is scoped by its SID.
	s.request(t, http.MethodPost, "/api/push/device/unregister", map[string]string{"deviceToken": "shared-device"}, old.AccessToken, "ios", "100", 200)
	for range 2 {
		s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken, "deviceToken": "shared-device"}, "", "ios", "100", 204)
	}
	// A matching already-revoked proof can finish legacy cleanup after a lost response.
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken, "deviceToken": "legacy-device"}, "", "ios", "100", 204)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE DEVICE_TOKEN='legacy-device'`)
	goldenCount(t, s.db, 3, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100 AND DEVICE_TOKEN='shared-device' AND SESSION_SID=?`, newSession.SID)
	targets, err := repo.ListDevices(100)
	if err != nil || len(targets) != 2 {
		t.Fatalf("live targets=%d err=%v", len(targets), err)
	}
	// Provider-invalid cleanup from a queued old delivery cannot remove the new owner.
	if err := repo.DeleteDeliveryTarget(100, model.PushDeliveryTarget{Platform: "android", DeviceToken: "shared-device", SessionID: old.SID}); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, s.db, 3, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN`)
}

func TestPushScopeHTTPRevocationAndExpirySuppressRetainedRows(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	old := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, old, "retained-device")
	pushScopeRow(t, s, 100, "legacy-device", nil)
	repo := repository.NewPushRepository(s.db)
	targets, err := repo.ListDevices(100)
	if err != nil || len(targets) != 1 {
		t.Fatalf("targets=%d err=%v", len(targets), err)
	}
	queued := targets[0]
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken}, "", "ios", "100", 204)
	goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN`)
	targets, err = repo.ListDevices(100)
	if err != nil || len(targets) != 0 {
		t.Fatalf("revoked targets=%d err=%v", len(targets), err)
	}
	current, err := repo.DeliveryTargetStillCurrent(100, queued)
	if err != nil || current {
		t.Fatalf("queued revoked current=%v err=%v", current, err)
	}
	seqs, err := repo.ListNoticeRecipientSeqs(0, 100)
	if err != nil || len(seqs) != 0 {
		t.Fatalf("notice recipients=%v err=%v", seqs, err)
	}
	live := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, live, "legacy-device")
	goldenExec(t, s.db, `UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET EXPIRES_AT=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE MRT_SID=?`, live.SID)
	targets, err = repo.ListDevices(100)
	if err != nil || len(targets) != 0 {
		t.Fatalf("expired targets=%d err=%v", len(targets), err)
	}
	expired := deferredSyntheticProof(t, 100, live.SID, live.JTI, time.Now().Add(-time.Minute).Unix())
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": expired, "deviceToken": "legacy-device"}, "", "ios", "100", 204)
	// 204 on expired proof is a no-op, not a claim of push-row cleanup; delivery remains suppressed.
	goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN`)
}

func TestPushScopeHTTPGlobalConsumedProofOneShotAndIsolation(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	old := pushScopeSession(t, s, 100)
	rotated, err := service.NewMobileSessionIssuer(s.deps.authService).Rotate(old.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	newer := pushScopeSession(t, s, 100)
	other := pushScopeSession(t, s, 101)
	pushScopeRegister(t, s, rotated, "old-device")
	pushScopeRegister(t, s, newer, "new-device")
	pushScopeRegister(t, s, other, "other-device")
	pushScopeRow(t, s, 100, "legacy-device", nil)
	pushScopeLegacyLog(t, s, 100)
	pushScopeLegacyLog(t, s, 101)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken}, "", "ios", "100", 204)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken}, "", "ios", "100", 401)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND (REVOKED_AT IS NULL OR MRT_REVOKED_AT IS NULL)`)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER_LOG WHERE USR_SEQ=100`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=101`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER_LOG WHERE USR_SEQ=101`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=101 AND REVOKED_AT IS NULL`)
	// A newer unrelated login is not covered by replaying a proof that already completed global logout.
	latest := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, latest, "latest-device")
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken}, "", "ios", "100", 401)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND REVOKED_AT IS NULL`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
}

func TestPushScopeHTTPGlobalInvalidProofAndIndependentLimiter(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	live := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, live, "live-device")
	cases := []string{
		live.AccessToken, live.RefreshToken + "x",
		deferredSyntheticProof(t, 101, live.SID, live.JTI, live.RefreshExpiresAt),
		deferredSyntheticProof(t, 100, strings.Repeat("f", 32), live.JTI, live.RefreshExpiresAt),
		deferredSyntheticProof(t, 100, live.SID, live.JTI, live.RefreshExpiresAt-1),
		deferredSyntheticProof(t, 100, live.SID, strings.Repeat("e", 32), live.RefreshExpiresAt),
		deferredSyntheticProof(t, 100, live.SID, live.JTI, time.Now().Add(-time.Minute).Unix()),
	}
	for _, proof := range cases {
		s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": proof}, "", "ios", "100", 401)
	}
	for n := len(cases); n < 10; n++ {
		s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": "invalid"}, "", "ios", "100", 401)
	}
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken}, "", "ios", "100", 429)
	// Endpoint buckets are independent: exhausted global attempts do not block single cleanup.
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": "invalid"}, "", "ios", "100", 401)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND REVOKED_AT IS NULL`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
}

func TestPushScopeHTTPGlobalCommitAndDeleteFailuresAtomic(t *testing.T) {
	fault := &atomic.Bool{}
	s := newGoldenServerWithDatabase(t, func(database *mariadb.Database) *sqlx.DB {
		name := fmt.Sprintf("push-scope-fault-%d", deferredFaultDriverSequence.Add(1))
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
	pushScopeRegister(t, s, live, "device")
	pushScopeLegacyLog(t, s, 100)
	assertUnchanged := func() {
		t.Helper()
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL`)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER_LOG WHERE USR_SEQ=100`)
	}
	fault.Store(true)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken}, "", "ios", "100", 500)
	if fault.Load() {
		t.Fatal("commit boundary not exercised")
	}
	assertUnchanged()
	goldenExec(t, s.db, `CREATE TRIGGER synthetic_push_delete_failure BEFORE DELETE ON ALUMNI_MOBILE_DEVICE_TOKEN FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic delete failure'`)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken}, "", "ios", "100", 500)
	assertUnchanged()
	// Single device cleanup shares the exact same transaction guarantee.
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken, "deviceToken": "device"}, "", "ios", "100", 500)
	assertUnchanged()
	goldenExec(t, s.db, `DROP TRIGGER synthetic_push_delete_failure`)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken}, "", "ios", "100", 204)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
}

func TestPushScopeActualMariaDBDelayedRegistrationCannotResurrectRevokedSID(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	old := pushScopeSession(t, s, 100)
	newer := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, old, "shared-device")
	tx, err := s.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var jti string
	if err := tx.Get(&jti, `SELECT MRT_JTI FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? FOR UPDATE`, old.JTI); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		result <- repository.NewPushRepository(s.db).RegisterDevice(100, model.PushDeviceRegistration{Platform: "android", DeviceToken: "shared-device", Locale: "ko-KR", SessionID: old.SID})
	}()
	<-started
	select {
	case err := <-result:
		t.Fatalf("old register bypassed held family lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	// Simulate deferred revocation holding the family lock before registration's current read.
	if _, err := tx.Exec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT=NOW(),MRT_REVOKED_AT=NOW() WHERE MRT_SID=?`, old.SID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE SESSION_SID=?`, old.SID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, repository.ErrRefreshTokenInvalid) {
			t.Fatalf("late register error=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("late register stuck")
	}
	pushScopeRegister(t, s, newer, "shared-device")
	s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": old.RefreshToken, "deviceToken": "shared-device"}, "", "ios", "100", 204)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE DEVICE_TOKEN='shared-device' AND SESSION_SID=?`, newer.SID)
}

func TestPushScopeHTTPConcurrentGlobalOneShot(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	live := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, live, "device")
	payload, _ := json.Marshal(map[string]string{"refreshToken": live.RefreshToken})
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, s.server.URL+allProofLogoutPath, bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			res, err := s.client.Do(req)
			if err != nil {
				statuses <- 0
				return
			}
			res.Body.Close()
			statuses <- res.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[204] != 1 || counts[401] != 1 {
		t.Fatalf("concurrent statuses=%v", counts)
	}
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
}

func TestPushScopeHTTPGlobalCancellationRollsBack(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	live := pushScopeSession(t, s, 100)
	pushScopeRegister(t, s, live, "device")
	pushScopeLegacyLog(t, s, 100)
	goldenExec(t, s.db, `CREATE TRIGGER synthetic_push_revoke_delay BEFORE UPDATE ON ALUMNI_MOBILE_REFRESH_TOKEN FOR EACH ROW SET @synthetic_delay=SLEEP(0.5)`)
	payload, _ := json.Marshal(map[string]string{"refreshToken": live.RefreshToken})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.server.URL+allProofLogoutPath, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if res != nil {
		res.Body.Close()
	}
	if err == nil {
		t.Fatal("cancelled request completed")
	}
	time.Sleep(600 * time.Millisecond)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND REVOKED_AT IS NULL`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER_LOG WHERE USR_SEQ=100`)
}

func TestPushScopeMigration085IsAdditiveIdempotent(t *testing.T) {
	migrations := postBaselineMigrations()
	database := mariadb.Start(t).NewDatabase(t, append(mariadb.ProdBaseline(t), migrations[:len(migrations)-1]...)...)
	goldenExec(t, database.DB, `INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN (USR_SEQ,PLATFORM,DEVICE_TOKEN,LOCALE) VALUES (100,'android','legacy','ko-KR')`)
	if err := repository.PushSessionSchemaReady(database.DB); err == nil {
		t.Fatal("old schema passed startup gate")
	}
	script, err := os.ReadFile(filepath.Join("..", "..", "migrations", "085_bind_push_device_to_mobile_session.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := database.DB.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	if err := repository.PushSessionSchemaReady(database.DB); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, database.DB, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100 AND DEVICE_TOKEN='legacy' AND SESSION_SID IS NULL`)
	goldenCount(t, database.DB, 1, `SELECT COUNT(DISTINCT INDEX_NAME) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_DEVICE_TOKEN' AND INDEX_NAME='IDX_MDT_ACCOUNT_SESSION'`)
}

func TestPushScopeHTTPGlobalBlockedBuildAndClosedBody(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	seedGoldenPolicy(t, s, "ios")
	live := pushScopeSession(t, s, 100)
	s.request(t, http.MethodGet, "/api/auth/me", nil, live.AccessToken, "ios", "99", 426)
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken, "deviceToken": "not-allowed"}, "", "ios", "99", 400)
	for _, device := range []string{"", strings.Repeat("a", 513), "space token", "\t", "é"} {
		s.request(t, http.MethodPost, singleProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken, "deviceToken": device}, "", "ios", "99", 400)
	}
	s.request(t, http.MethodPost, allProofLogoutPath, map[string]string{"refreshToken": live.RefreshToken}, "", "ios", "99", 204)
}

func pushScopeRawLogout(s *goldenServer, path, proof, device string) (int, error) {
	body := map[string]string{"refreshToken": proof}
	if device != "" {
		body["deviceToken"] = device
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, s.server.URL+path, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	return res.StatusCode, nil
}

func TestPushScopeActualMariaDBRegistrationVersusHTTPProofRevocation(t *testing.T) {
	for _, global := range []bool{false, true} {
		t.Run(fmt.Sprintf("global=%v", global), func(t *testing.T) {
			s := newGoldenServer(t)
			pushScopeSeed(t, s)
			old := pushScopeSession(t, s, 100)
			pushScopeRegister(t, s, old, "racing-device")
			tx, err := s.db.Beginx()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var jti string
			if err := tx.Get(&jti, `SELECT MRT_JTI FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE MRT_JTI=? FOR UPDATE`, old.JTI); err != nil {
				t.Fatal(err)
			}
			registration := make(chan error, 1)
			logout := make(chan int, 1)
			path := singleProofLogoutPath
			device := "racing-device"
			if global {
				path = allProofLogoutPath
				device = ""
			}
			go func() {
				registration <- repository.NewPushRepository(s.db).RegisterDevice(100, model.PushDeviceRegistration{Platform: "android", DeviceToken: "racing-device", Locale: "ko-KR", SessionID: old.SID})
			}()
			go func() {
				status, err := pushScopeRawLogout(s, path, old.RefreshToken, device)
				if err != nil {
					logout <- 0
				} else {
					logout <- status
				}
			}()
			// Both writes must wait on the exact family row held by a third transaction.
			select {
			case status := <-logout:
				t.Fatalf("logout bypassed family lock status=%d", status)
			case <-time.After(100 * time.Millisecond):
			}
			select {
			case err := <-registration:
				t.Fatalf("registration bypassed family lock err=%v", err)
			default:
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-registration:
				if err != nil && !errors.Is(err, repository.ErrRefreshTokenInvalid) {
					t.Fatalf("registration err=%v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("registration deadlocked")
			}
			select {
			case status := <-logout:
				if status != 204 {
					t.Fatalf("logout status=%d", status)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("logout deadlocked")
			}
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=100 AND REVOKED_AT IS NULL`)
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=100`)
		})
	}
}

func TestPushScopeActualMariaDBNewRegistrationVersusOldProofCleanup(t *testing.T) {
	s := newGoldenServer(t)
	pushScopeSeed(t, s)
	for n := 0; n < 3; n++ {
		old := pushScopeSession(t, s, 100)
		newer := pushScopeSession(t, s, 100)
		token := fmt.Sprintf("race-new-%d", n)
		pushScopeRegister(t, s, old, token)
		start := make(chan struct{})
		registered := make(chan error, 1)
		revoked := make(chan int, 1)
		go func() {
			<-start
			registered <- repository.NewPushRepository(s.db).RegisterDevice(100, model.PushDeviceRegistration{Platform: "android", DeviceToken: token, Locale: "ko-KR", SessionID: newer.SID})
		}()
		go func() {
			<-start
			status, err := pushScopeRawLogout(s, singleProofLogoutPath, old.RefreshToken, token)
			if err != nil {
				revoked <- 0
			} else {
				revoked <- status
			}
		}()
		close(start)
		if err := <-registered; err != nil {
			t.Fatal(err)
		}
		if status := <-revoked; status != 204 {
			t.Fatalf("old revoke status=%d", status)
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE DEVICE_TOKEN=? AND SESSION_SID=?`, token, newer.SID)
	}
}
