package repository

import (
	"errors"
)

// AnonymizeAccountForDeletion keeps its legacy name. Account withdrawal is
// represented by WEO_MEMBER.USR_STATUS = 'AAA'; any root-role companion is
// revoked atomically with that status change.
func (r *AuthRepository) AnonymizeAccountForDeletion(usrSeq int) error {
	affected, err := updateMemberStatusAndAdminRole(r.DB, usrSeq, "AAA")
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("account deletion did not update exactly one member")
	}
	return nil
}

const revocationActionAccountDelete = "ACCOUNT_DELETE"

func (r *AuthRepository) CompleteAccountDeletionRevocation(usrSeq int, provider string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		DELETE FROM ALUMNI_SOCIAL_CREDENTIAL
		WHERE USR_SEQ = ? AND PROVIDER = ?
	`, usrSeq, provider); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'DELIVERED', UPDATED_AT = NOW(), LAST_ERROR = NULL
		WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = ? AND STATUS IN ('PENDING','PROCESSING','REVOKED')
	`, usrSeq, provider, revocationActionAccountDelete); err != nil {
		return err
	}
	return tx.Commit()
}
