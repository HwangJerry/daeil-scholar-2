// social_revocation_repo_test.go — Verifies the claim/finalize/failure
// bookkeeping the social revocation worker depends on.
package repository_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

// TestClaimDueSocialRevocations proves the query selects both PENDING
// (upstream not yet revoked) and REVOKED (upstream revoked, local
// finalization pending) rows, and that claiming is a two-step
// UPDATE-then-SELECT keyed on the caller's claim token.
func TestClaimDueSocialRevocations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	now := time.Now()

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX\s+SET CLAIM_TOKEN = \?, UPDATED_AT = NOW\(\)\s+WHERE STATUS IN \('PENDING', 'REVOKED'\)\s+AND NEXT_ATTEMPT_AT <= NOW\(\)\s+AND \(CLAIM_TOKEN IS NULL OR UPDATED_AT <= NOW\(\) - INTERVAL \? SECOND\)`).
		WithArgs("worker-token", 300, 20).
		WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectQuery(`SELECT OUTBOX_ID, USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT.*WHERE CLAIM_TOKEN = \?`).
		WithArgs("worker-token").
		WillReturnRows(sqlmock.NewRows([]string{
			"OUTBOX_ID", "USR_SEQ", "PROVIDER", "ACTION", "STATUS", "ATTEMPT_COUNT",
			"NEXT_ATTEMPT_AT", "LAST_ERROR", "CREATED_AT", "UPDATED_AT",
		}).
			AddRow(1, 42, "KT", "DISCONNECT", "PENDING", 0, now, "", now, now).
			AddRow(2, 43, "AP", "ACCOUNT_DELETE", "REVOKED", 1, now, "delete failed", now, now))

	entries, err := repo.ClaimDueSocialRevocations("worker-token", 5*time.Minute, 20)
	if err != nil {
		t.Fatalf("ClaimDueSocialRevocations() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].OutboxID != 1 || entries[0].USRSeq != 42 || entries[0].Provider != "KT" || entries[0].Status != "PENDING" {
		t.Fatalf("unexpected entry[0]: %+v", entries[0])
	}
	if entries[1].OutboxID != 2 || entries[1].Action != "ACCOUNT_DELETE" || entries[1].Status != "REVOKED" {
		t.Fatalf("unexpected entry[1]: %+v", entries[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkSocialRevocationRevoked(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS = 'REVOKED'`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationRevoked(1); err != nil {
		t.Fatalf("MarkSocialRevocationRevoked() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkSocialRevocationSucceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET CLAIM_TOKEN = NULL`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationSucceeded(1); err != nil {
		t.Fatalf("MarkSocialRevocationSucceeded() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkSocialRevocationFailedRetriesUntilCap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	next := time.Now().Add(time.Minute)

	// Below the attempt cap: stays at the caller-supplied retry status.
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs("PENDING", 3, "boom", next, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationFailed(1, "boom", 3, 10, next, "PENDING"); err != nil {
		t.Fatalf("MarkSocialRevocationFailed() error = %v", err)
	}

	// At the attempt cap: becomes terminal FAILED regardless of retryStatus.
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs("FAILED", 10, "boom", next, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationFailed(1, "boom", 10, 10, next, "PENDING"); err != nil {
		t.Fatalf("MarkSocialRevocationFailed() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestMarkSocialRevocationFailedPreservesRevokedRetryStatus proves that when
// the upstream provider was already revoked (only local finalization
// failed), a retry-under-cap is reset to REVOKED rather than PENDING, so the
// worker won't call the provider's unlink/revoke API a second time on retry.
func TestMarkSocialRevocationFailedPreservesRevokedRetryStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	next := time.Now().Add(time.Minute)

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs("REVOKED", 2, "finalize failed", next, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationFailed(1, "finalize failed", 2, 10, next, "REVOKED"); err != nil {
		t.Fatalf("MarkSocialRevocationFailed() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestFinalizeSocialDisconnectAdvancesActiveDisconnect covers the normal
// case: WEO_MEMBER_SOCIAL is still DISCONNECTING when the worker finalizes.
func TestFinalizeSocialDisconnectAdvancesActiveDisconnect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL\s+SET NMS_STATUS = 'FINALIZE_PENDING'\s+WHERE USR_SEQ = \? AND NMS_GATE = \? AND NMS_STATUS = 'DISCONNECTING'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.FinalizeSocialDisconnect(42, "KT"); err != nil {
		t.Fatalf("FinalizeSocialDisconnect() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestFinalizeSocialDisconnectToleratesAlreadyFinalizePending proves a retry
// after a crash between MarkSocialDisconnectRevoked and DeleteSocialConnection
// succeeds instead of erroring, since the link is already in the expected
// intermediate state.
func TestFinalizeSocialDisconnectToleratesAlreadyFinalizePending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL\s+SET NMS_STATUS = 'FINALIZE_PENDING'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT NMS_STATUS FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"NMS_STATUS"}).AddRow("FINALIZE_PENDING"))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.FinalizeSocialDisconnect(42, "KT"); err != nil {
		t.Fatalf("FinalizeSocialDisconnect() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestFinalizeSocialDisconnectToleratesAlreadyDeleted proves a retry after
// the link row itself was already removed by a prior finalize attempt
// succeeds (idempotent full completion) instead of erroring.
func TestFinalizeSocialDisconnectToleratesAlreadyDeleted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL\s+SET NMS_STATUS = 'FINALIZE_PENDING'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT NMS_STATUS FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.FinalizeSocialDisconnect(42, "KT"); err != nil {
		t.Fatalf("FinalizeSocialDisconnect() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestFinalizeSocialDisconnectRejectsUnexpectedState proves a genuinely
// anomalous link state (neither DISCONNECTING, FINALIZE_PENDING, nor
// deleted) is surfaced as an error rather than silently overwritten.
func TestFinalizeSocialDisconnectRejectsUnexpectedState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL\s+SET NMS_STATUS = 'FINALIZE_PENDING'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT NMS_STATUS FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"NMS_STATUS"}).AddRow("ACTIVE"))

	if err := repo.FinalizeSocialDisconnect(42, "KT"); err == nil {
		t.Fatal("expected an error for an unexpected link state, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
