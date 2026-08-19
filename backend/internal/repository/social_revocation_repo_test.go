package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// TestClaimDueSocialRevocations proves ClaimDueSocialRevocations does both
// halves of claiming: an UPDATE that stamps CLAIM_TOKEN onto due PENDING/
// REVOKED rows not already claimed (or whose claim has gone stale), then a
// SELECT of exactly the rows carrying that token. Selecting both PENDING and
// REVOKED matters - if it only matched STATUS = 'PENDING', a row stuck in
// REVOKED after a crash would never be picked back up by the worker.
func TestClaimDueSocialRevocations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
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
			AddRow(2, 43, "AP", "DISCONNECT", "REVOKED", 1, now, "delete failed", now, now))

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
	if entries[1].OutboxID != 2 || entries[1].Status != "REVOKED" {
		t.Fatalf("unexpected entry[1]: %+v", entries[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestMarkSocialRevocationRevoked proves the durable REVOKED checkpoint is
// written as its own UPDATE - the worker relies on this happening BEFORE
// local cleanup is attempted, so that a failure at any later step never
// causes a retry to call the provider's revoke/unlink API again.
func TestMarkSocialRevocationRevoked(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

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
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS = 'DONE'`).
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
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
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
// the upstream provider was already revoked (only local cleanup failed), a
// retry-under-cap is reset to REVOKED rather than PENDING, so the worker
// won't call the provider's unlink/revoke API a second time on retry.
func TestMarkSocialRevocationFailedPreservesRevokedRetryStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	next := time.Now().Add(time.Minute)

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs("REVOKED", 2, "delete failed", next, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationFailed(1, "delete failed", 2, 10, next, "REVOKED"); err != nil {
		t.Fatalf("MarkSocialRevocationFailed() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestMarkSocialRevocationFailedTerminatesRevokedAtCap proves the
// attempt-cap-exceeded transition to terminal FAILED overrides retryStatus
// even when retrying is "REVOKED" - repeated local-cleanup failures must
// eventually stop retrying instead of looping forever.
func TestMarkSocialRevocationFailedTerminatesRevokedAtCap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	next := time.Now().Add(time.Minute)

	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs("FAILED", 10, "delete failed", next, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkSocialRevocationFailed(1, "delete failed", 10, 10, next, "REVOKED"); err != nil {
		t.Fatalf("MarkSocialRevocationFailed() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestDeleteSocialConnection locks in the behavior the revocation worker now
// depends on to finish a deferred disconnect: after a successful upstream
// revoke, it must delete both the WEO_MEMBER_SOCIAL link and the encrypted
// ALUMNI_SOCIAL_CREDENTIAL row within a single transaction.
func TestDeleteSocialConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.DeleteSocialConnection(42, "KT"); err != nil {
		t.Fatalf("DeleteSocialConnection() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
