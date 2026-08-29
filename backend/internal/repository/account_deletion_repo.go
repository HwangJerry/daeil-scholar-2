package repository

import (
	"errors"
)

// AnonymizeAccountForDeletion keeps its legacy name, but account withdrawal is
// currently represented only by WEO_MEMBER.USR_STATUS = 'AAA'.
func (r *AuthRepository) AnonymizeAccountForDeletion(usrSeq int) error {
	result, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_STATUS = 'AAA' WHERE USR_SEQ = ?`, usrSeq)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
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
