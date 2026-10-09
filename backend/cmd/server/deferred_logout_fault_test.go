package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

var deferredFaultDriverSequence atomic.Int64

type deferredCommitFaultDriver struct{ fail *atomic.Bool }

func (d *deferredCommitFaultDriver) Open(name string) (driver.Conn, error) {
	c, err := (&mysql.MySQLDriver{}).Open(name)
	if err != nil {
		return nil, err
	}
	return &deferredCommitFaultConn{Conn: c, fail: d.fail}, nil
}

type deferredCommitFaultConn struct {
	driver.Conn
	fail *atomic.Bool
}

func (c *deferredCommitFaultConn) Begin() (driver.Tx, error) {
	tx, err := c.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return &deferredCommitFaultTx{Tx: tx, fail: c.fail}, nil
}
func (c *deferredCommitFaultConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	tx, err := c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &deferredCommitFaultTx{Tx: tx, fail: c.fail}, nil
}

type deferredCommitFaultTx struct {
	driver.Tx
	fail *atomic.Bool
}

func (tx *deferredCommitFaultTx) Commit() error {
	if tx.fail.Swap(false) {
		_ = tx.Tx.Rollback()
		return errors.New("synthetic commit-boundary rollback")
	}
	return tx.Tx.Commit()
}

func TestDeferredLogoutHTTPCommitFailureRollsBackActualMariaDB(t *testing.T) {
	fault := &atomic.Bool{}
	s := newGoldenServerWithDatabase(t, func(database *mariadb.Database) *sqlx.DB {
		name := fmt.Sprintf("deferred-mariadb-fault-%d", deferredFaultDriverSequence.Add(1))
		sql.Register(name, &deferredCommitFaultDriver{fail: fault})
		db, err := sqlx.Open(name, database.DSN)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	})
	seedGoldenMember(t, s.db)
	user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	fault.Store(true)
	body := s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 500)
	assertGoldenError(t, body, "LOGOUT_FAILED")
	if fault.Load() {
		t.Fatal("commit fault interceptor was not exercised")
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL`, goldenMemberID)
	s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 204)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL`, goldenMemberID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=?`, goldenMemberID)
}

func TestDeferredLogoutHTTPUpdateFailureAndCancellationRollback(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(goldenMemberID)
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `CREATE TRIGGER synthetic_revoke_update_failure BEFORE UPDATE ON ALUMNI_MOBILE_REFRESH_TOKEN FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic revocation failure'`)
	body := s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 500)
	assertGoldenError(t, body, "LOGOUT_FAILED")
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL`, goldenMemberID)
	goldenExec(t, s.db, `DROP TRIGGER synthetic_revoke_update_failure`)
	goldenExec(t, s.db, `CREATE TRIGGER synthetic_revoke_update_delay BEFORE UPDATE ON ALUMNI_MOBILE_REFRESH_TOKEN FOR EACH ROW SET @synthetic_revoke_delay=SLEEP(0.5)`)
	payload, _ := json.Marshal(map[string]string{"refreshToken": session.RefreshToken})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.server.URL+"/api/auth/logout/deferred", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("cancelled HTTP unexpectedly completed")
	}
	time.Sleep(600 * time.Millisecond)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_MOBILE_REFRESH_TOKEN WHERE USR_SEQ=? AND REVOKED_AT IS NULL AND MRT_REVOKED_AT IS NULL`, goldenMemberID)
	goldenExec(t, s.db, `DROP TRIGGER synthetic_revoke_update_delay`)
	s.request(t, http.MethodPost, "/api/auth/logout/deferred", map[string]string{"refreshToken": session.RefreshToken}, "", "ios", "100", 204)
}
