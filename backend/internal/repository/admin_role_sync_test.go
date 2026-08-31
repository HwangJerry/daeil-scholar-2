package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestSyncAdminRoleForStatusChangeUpsertsOnPromotion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "CCC", Valid: true}, "ZZZ"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAdminRoleForStatusChangeDeletesOnDemotion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM ALUMNI_ADMIN_ROLE[\s\S]*ADMIN_ROLE = \?`).
		WithArgs(42, rootAdminRole).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "ZZZ", Valid: true}, "CCC"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncAdminRoleForStatusChangeDoesNothingWhenStatusIsUnchanged(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncAdminRoleForStatusChangeTx(tx, 42, sql.NullString{String: "ZZZ", Valid: true}, "ZZZ"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSyncVerificationForStatusChangeUpsertsLegacyStatusMapping(t *testing.T) {
	tests := []struct {
		name                   string
		oldStatus              string
		newStatus              string
		wantVerificationStatus model.VerificationStatus
		wantRejectionReason    any
		wantApprovedFields     int
	}{
		{
			name:                   "BAA rejected",
			oldStatus:              pendingMemberStatus,
			newStatus:              rejectedMemberStatus,
			wantVerificationStatus: model.VerificationRejected,
			wantRejectionReason:    legacyMemberStatusRejectionReason,
		},
		{
			name:                   "BBB pending",
			oldStatus:              rejectedMemberStatus,
			newStatus:              pendingMemberStatus,
			wantVerificationStatus: model.VerificationPending,
		},
		{
			name:                   "CCC approved",
			oldStatus:              pendingMemberStatus,
			newStatus:              approvedMemberStatus,
			wantVerificationStatus: model.VerificationApproved,
			wantApprovedFields:     1,
		},
		{
			name:                   "ZZZ approved",
			oldStatus:              approvedMemberStatus,
			newStatus:              rootAdminMemberStatus,
			wantVerificationStatus: model.VerificationApproved,
			wantApprovedFields:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := "29"
			dept := "영어"
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION[\s\S]*VALUES[\s\S]*NULLIF\(TRIM\(\?\), ''\)[\s\S]*ON DUPLICATE KEY UPDATE`).
				WithArgs(
					42,
					tt.wantVerificationStatus,
					fn,
					dept,
					tt.wantRejectionReason,
					tt.wantApprovedFields,
					fn,
					tt.wantApprovedFields,
					dept,
				).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
			if err != nil {
				t.Fatal(err)
			}
			oldStatus := sql.NullString{String: tt.oldStatus, Valid: true}
			if err := syncVerificationForStatusChangeTx(tx, 42, oldStatus, tt.newStatus, fn, dept); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSyncVerificationForStatusChangeIgnoresUnchangedAndUnmappedStatuses(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus sql.NullString
		newStatus string
	}{
		{
			name:      "unchanged mapped status",
			oldStatus: sql.NullString{String: approvedMemberStatus, Valid: true},
			newStatus: approvedMemberStatus,
		},
		{
			name:      "changed unmapped status",
			oldStatus: sql.NullString{String: approvedMemberStatus, Valid: true},
			newStatus: "AAA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			mock.ExpectBegin()
			mock.ExpectCommit()

			tx, err := sqlx.NewDb(db, "sqlmock").Beginx()
			if err != nil {
				t.Fatal(err)
			}
			if err := syncVerificationForStatusChangeTx(tx, 42, tt.oldStatus, tt.newStatus, "", ""); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUpdateMemberStatusAndAdminRoleRollsBackPromotionWhenRoleUpsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	roleSyncErr := errors.New("role upsert failed")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_STATUS", "USR_FN", "USR_DEPT"}).AddRow("CCC", "29", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS = \? WHERE USR_SEQ = \?`).
		WithArgs(rootAdminMemberStatus, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnError(roleSyncErr)
	mock.ExpectRollback()

	_, err = updateMemberStatusAndAdminRole(sqlx.NewDb(db, "sqlmock"), 42, rootAdminMemberStatus)
	if !errors.Is(err, roleSyncErr) {
		t.Fatalf("error = %v, want %v", err, roleSyncErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateMemberStatusAndAdminRoleRollsBackDemotionWhenRoleDeleteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	roleSyncErr := errors.New("role delete failed")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_STATUS", "USR_FN", "USR_DEPT"}).AddRow(rootAdminMemberStatus, "29", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS = \? WHERE USR_SEQ = \?`).
		WithArgs("CCC", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_ADMIN_ROLE[\s\S]*ADMIN_ROLE = \?`).
		WithArgs(42, rootAdminRole).
		WillReturnError(roleSyncErr)
	mock.ExpectRollback()

	_, err = updateMemberStatusAndAdminRole(sqlx.NewDb(db, "sqlmock"), 42, "CCC")
	if !errors.Is(err, roleSyncErr) {
		t.Fatalf("error = %v, want %v", err, roleSyncErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateMemberStatusAndAdminRoleRollsBackWhenVerificationSyncFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	verificationSyncErr := errors.New("verification upsert failed")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_STATUS", "USR_FN", "USR_DEPT"}).AddRow(pendingMemberStatus, "29", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER SET USR_STATUS = \? WHERE USR_SEQ = \?`).
		WithArgs(approvedMemberStatus, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_ADMIN_ROLE[\s\S]*ADMIN_ROLE = \?`).
		WithArgs(42, rootAdminRole).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, model.VerificationApproved, "29", "영어", nil, 1, "29", 1, "영어").
		WillReturnError(verificationSyncErr)
	mock.ExpectRollback()

	_, err = updateMemberStatusAndAdminRole(sqlx.NewDb(db, "sqlmock"), 42, approvedMemberStatus)
	if !errors.Is(err, verificationSyncErr) {
		t.Fatalf("error = %v, want %v", err, verificationSyncErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertRootAdminMemberCreatesRoleWithMemberAsAuditActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ, USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs("admin").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WithArgs("admin", "hash", "관리자").
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION[\s\S]*VALUES[\s\S]*NULLIF\(TRIM\(\?\), ''\)`).
		WithArgs(42, model.VerificationApproved, "", "", nil, 1, "", 1, "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	usrSeq, err := repo.UpsertRootAdminMember("admin", "hash", "관리자")
	if err != nil {
		t.Fatal(err)
	}
	if usrSeq != 42 {
		t.Fatalf("usrSeq = %d, want 42", usrSeq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertRootAdminMemberPromotesExistingMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ, USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_STATUS", "USR_FN", "USR_DEPT"}).AddRow(42, "CCC", "29", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER[\s\S]*USR_STATUS = 'ZZZ'`).
		WithArgs("new-hash", "관리자", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, model.VerificationApproved, "29", "영어", nil, 1, "29", 1, "영어").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	usrSeq, err := repo.UpsertRootAdminMember("admin", "new-hash", "관리자")
	if err != nil {
		t.Fatal(err)
	}
	if usrSeq != 42 {
		t.Fatalf("usrSeq = %d, want 42", usrSeq)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertRootAdminMemberRollsBackExistingMemberUpdateWhenRoleUpsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock"))
	roleSyncErr := errors.New("role upsert failed")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ, USR_STATUS,[\s\S]*USR_FN[\s\S]*USR_DEPT[\s\S]*FOR UPDATE`).
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_STATUS", "USR_FN", "USR_DEPT"}).AddRow(42, "CCC", "29", "영어"))
	mock.ExpectExec(`UPDATE WEO_MEMBER[\s\S]*USR_STATUS = 'ZZZ'`).
		WithArgs("new-hash", "관리자", 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE[\s\S]*ON DUPLICATE KEY UPDATE`).
		WithArgs(42, rootAdminRole, 42, 42).
		WillReturnError(roleSyncErr)
	mock.ExpectRollback()

	_, err = repo.UpsertRootAdminMember("admin", "new-hash", "관리자")
	if !errors.Is(err, roleSyncErr) {
		t.Fatalf("error = %v, want %v", err, roleSyncErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
