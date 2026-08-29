package repository

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

const (
	rootAdminMemberStatus = "ZZZ"
	rootAdminRole         = "root"
)

func syncAdminRoleForMemberInsertTx(tx *sqlx.Tx, usrSeq int, status string) error {
	if status != rootAdminMemberStatus {
		return nil
	}
	return upsertRootAdminRoleTx(tx, usrSeq)
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
// canonical root-role companion atomically.
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
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}
