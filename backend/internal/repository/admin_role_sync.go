package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

const (
	rejectedMemberStatus              = "BAA"
	pendingMemberStatus               = "BBB"
	approvedMemberStatus              = "CCC"
	rootAdminMemberStatus             = "ZZZ"
	rootAdminRole                     = "root"
	legacyMemberStatusRejectionReason = "기존 회원 상태로 인해 반려됨"
)

func syncAdminRoleForMemberInsertTx(tx *sqlx.Tx, usrSeq int, status string) error {
	if status != rootAdminMemberStatus {
		return nil
	}
	return upsertRootAdminRoleTx(tx, usrSeq)
}

// syncVerificationForMemberInsertTx mirrors the ALUMNI_VERIFICATION portion of
// TRG_WEO_MEMBER_AUTH_PRINCIPAL_INSERT for a newly inserted member.
func syncVerificationForMemberInsertTx(tx *sqlx.Tx, usrSeq int, status, fn, dept string) error {
	verificationStatus, rejectionReason, approvedAcademicFields, shouldSync := verificationValuesForMemberStatus(status)
	if !shouldSync {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO ALUMNI_VERIFICATION (
			USR_SEQ, STATUS, COHORT, DEPARTMENT, REJECTION_REASON,
			APPROVED_COHORT, APPROVED_DEPARTMENT, CREATED_AT, UPDATED_AT
		) VALUES (
			?, ?, NULLIF(TRIM(?), ''), NULLIF(TRIM(?), ''), ?,
			CASE WHEN ? = 1 THEN NULLIF(TRIM(?), '') ELSE NULL END,
			CASE WHEN ? = 1 THEN NULLIF(TRIM(?), '') ELSE NULL END,
			NOW(), NOW()
		)
	`,
		usrSeq,
		verificationStatus,
		fn,
		dept,
		rejectionReason,
		approvedAcademicFields,
		fn,
		approvedAcademicFields,
		dept,
	)
	return err
}

// syncAdminRoleForStatusChangeTx mirrors the ALUMNI_ADMIN_ROLE portion of
// TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE. sql.NullString preserves the trigger's
// NULL-safe OLD/NEW comparison even though current application writes are
// non-NULL.
func syncAdminRoleForStatusChangeTx(tx *sqlx.Tx, usrSeq int, oldStatus sql.NullString, newStatus string) error {
	statusUnchanged := oldStatus.Valid && oldStatus.String == newStatus
	if statusUnchanged {
		return nil
	}
	if newStatus == rootAdminMemberStatus {
		return upsertRootAdminRoleTx(tx, usrSeq)
	}
	_, err := tx.Exec(`
		DELETE FROM ALUMNI_ADMIN_ROLE
		WHERE USR_SEQ = ? AND ADMIN_ROLE = ?
	`, usrSeq, rootAdminRole)
	return err
}

// syncVerificationForStatusChangeTx mirrors the ALUMNI_VERIFICATION portion of
// TRG_WEO_MEMBER_AUTH_PRINCIPAL_UPDATE. The member row has already been updated,
// so the INSERT ... SELECT reads the trigger's NEW.USR_FN and NEW.USR_DEPT values.
func syncVerificationForStatusChangeTx(tx *sqlx.Tx, usrSeq int, oldStatus sql.NullString, newStatus string) error {
	statusUnchanged := oldStatus.Valid && oldStatus.String == newStatus
	if statusUnchanged {
		return nil
	}

	verificationStatus, rejectionReason, approvedAcademicFields, shouldSync := verificationValuesForMemberStatus(newStatus)
	if !shouldSync {
		return nil
	}

	_, err := tx.Exec(`
		INSERT INTO ALUMNI_VERIFICATION (
			USR_SEQ, STATUS, COHORT, DEPARTMENT, REJECTION_REASON,
			APPROVED_COHORT, APPROVED_DEPARTMENT, CREATED_AT, UPDATED_AT
		)
		SELECT
			USR_SEQ,
			?,
			NULLIF(TRIM(USR_FN), ''),
			NULLIF(TRIM(USR_DEPT), ''),
			?,
			CASE WHEN ? = 1 THEN NULLIF(TRIM(USR_FN), '') ELSE NULL END,
			CASE WHEN ? = 1 THEN NULLIF(TRIM(USR_DEPT), '') ELSE NULL END,
			NOW(),
			NOW()
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		ON DUPLICATE KEY UPDATE
			STATUS = VALUES(STATUS),
			COHORT = VALUES(COHORT),
			DEPARTMENT = VALUES(DEPARTMENT),
			REJECTION_REASON = VALUES(REJECTION_REASON),
			APPROVED_COHORT = VALUES(APPROVED_COHORT),
			APPROVED_DEPARTMENT = VALUES(APPROVED_DEPARTMENT),
			UPDATED_AT = NOW()
	`, verificationStatus, rejectionReason, approvedAcademicFields, approvedAcademicFields, usrSeq)
	return err
}

func verificationValuesForMemberStatus(status string) (model.VerificationStatus, any, int, bool) {
	switch status {
	case rejectedMemberStatus:
		return model.VerificationRejected, legacyMemberStatusRejectionReason, 0, true
	case pendingMemberStatus:
		return model.VerificationPending, nil, 0, true
	case approvedMemberStatus, rootAdminMemberStatus:
		return model.VerificationApproved, nil, 1, true
	default:
		return "", nil, 0, false
	}
}

func upsertRootAdminRoleTx(tx *sqlx.Tx, usrSeq int) error {
	_, err := tx.Exec(`
		INSERT INTO ALUMNI_ADMIN_ROLE (
			USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT, CREATED_BY, UPDATED_BY
		) VALUES (?, ?, NOW(), NOW(), ?, ?)
		ON DUPLICATE KEY UPDATE
			ADMIN_ROLE = VALUES(ADMIN_ROLE), UPDATED_AT = NOW(), UPDATED_BY = VALUES(UPDATED_BY)
	`, usrSeq, rootAdminRole, usrSeq, usrSeq)
	return err
}

// updateMemberStatusAndAdminRole applies the legacy status update and its
// canonical authorization companions atomically.
func updateMemberStatusAndAdminRole(db *sqlx.DB, usrSeq int, newStatus string) (int64, error) {
	tx, err := db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var oldStatus sql.NullString
	memberExists := true
	if err := tx.Get(&oldStatus, `
		SELECT USR_STATUS
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		FOR UPDATE
	`, usrSeq); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		memberExists = false
	}

	result, err := tx.Exec(`UPDATE WEO_MEMBER SET USR_STATUS = ? WHERE USR_SEQ = ?`, newStatus, usrSeq)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if memberExists && affected > 0 {
		if err := syncAdminRoleForStatusChangeTx(tx, usrSeq, oldStatus, newStatus); err != nil {
			return 0, err
		}
		if err := syncVerificationForStatusChangeTx(tx, usrSeq, oldStatus, newStatus); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}
