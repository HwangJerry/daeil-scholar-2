package repository

import (
	"errors"
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
)

// AnonymizeAccountForDeletion performs every local account-withdrawal effect in
// one transaction. Provider credentials are retained only while their durable
// revocation outbox item is pending.
func (r *AuthRepository) AnonymizeAccountForDeletion(usrSeq int) ([]string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	phoneClaimsEnabled, err := r.phoneClaimsEnabledTx(tx)
	if err != nil {
		return nil, err
	}

	var lockedAccount struct {
		AccountID int    `db:"USR_SEQ"`
		Phone     string `db:"USR_PHONE"`
	}
	if err := tx.Get(&lockedAccount, `
		SELECT USR_SEQ, COALESCE(USR_PHONE, '') AS USR_PHONE
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		FOR UPDATE
	`, usrSeq); err != nil {
		return nil, err
	}
	donorPhone := model.NormalizePhoneNumber(lockedAccount.Phone).String()

	var providers []string
	if err := tx.Select(&providers, `
		SELECT NMS_GATE
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ?
		ORDER BY NMS_GATE
		FOR UPDATE
	`, usrSeq); err != nil {
		return nil, err
	}

	// Known limitation: unlinking identities can merge donor keys without refreshing donorCount; tracked as a follow-up.
	statements := []string{
		`UPDATE WEO_ORDER o
		 JOIN WEO_MEMBER m ON m.USR_SEQ = ?
		 SET o.O_DONOR_NAME = COALESCE(NULLIF(o.O_DONOR_NAME, ''), NULLIF(m.USR_NAME, '')),
		     o.O_DONOR_PHONE = COALESCE(
		       NULLIF(o.O_DONOR_PHONE, ''),
		       NULLIF(?, '')
		     ),
		     o.O_DONOR_COHORT = COALESCE(NULLIF(o.O_DONOR_COHORT, ''), NULLIF(m.USR_FN, '')),
		     o.O_DONOR_DEPARTMENT = COALESCE(NULLIF(o.O_DONOR_DEPARTMENT, ''), NULLIF(m.USR_DEPT, '')),
		     o.O_LEGAL_RETENTION_UNTIL = CASE
		       WHEN o.O_LEGAL_RETENTION_UNTIL IS NULL
		         OR o.O_LEGAL_RETENTION_UNTIL < DATE_ADD(COALESCE(o.O_DONATION_DATE, DATE(o.O_REGDATE), CURRENT_DATE), INTERVAL 5 YEAR)
		       THEN DATE_ADD(COALESCE(o.O_DONATION_DATE, DATE(o.O_REGDATE), CURRENT_DATE), INTERVAL 5 YEAR)
		       ELSE o.O_LEGAL_RETENTION_UNTIL
		     END,
		     o.O_ACCOUNT_USR_SEQ = NULL,
		     o.O_ACCOUNT_UNLINKED_AT = NOW(),
		     o.USR_SEQ = 0
		 WHERE o.O_ACCOUNT_USR_SEQ = m.USR_SEQ OR o.USR_SEQ = m.USR_SEQ`,
		`UPDATE ALUMNI_MESSAGE
		 SET AM_SENDER_ACCOUNT_SEQ = NULL,
		     AM_SENDER_ANONYMIZED_YN = 'Y',
		     AM_CLIENT_MESSAGE_ID = NULL,
		     AM_SENDER_SEQ = 0
		 WHERE AM_SENDER_ACCOUNT_SEQ = ? OR AM_SENDER_SEQ = ?`,
		`UPDATE ALUMNI_MESSAGE
		 SET AM_RECVR_ACCOUNT_SEQ = NULL,
		     AM_RECVR_ANONYMIZED_YN = 'Y',
		     AM_RECVR_SEQ = 0
		 WHERE AM_RECVR_ACCOUNT_SEQ = ? OR AM_RECVR_SEQ = ?`,
		`UPDATE AUTH_SESSION_FAMILY
		 SET STATUS = 'REVOKED', REVOKED_AT = COALESCE(REVOKED_AT, NOW()),
		     REVOKE_REASON_CODE = COALESCE(REVOKE_REASON_CODE, 'ACCOUNT_WITHDRAWN'), UPDATED_AT = NOW()
		 WHERE ACCOUNT_ID = ? AND STATUS = 'ACTIVE'`,
		`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		 SET REVOKED_AT = COALESCE(REVOKED_AT, NOW())
		 WHERE USR_SEQ = ?`,
		`DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?`,
		`DELETE FROM ALUMNI_PUSH_DEVICE WHERE USR_SEQ = ?`,
		`DELETE FROM ALUMNI_PUSH_PREFERENCE WHERE USR_SEQ = ?`,
		`DELETE FROM ALUMNI_MEMBER_BLOCK WHERE BLOCKER_USR_SEQ = ? OR BLOCKED_USR_SEQ = ?`,
		`DELETE FROM ALUMNI_USER_TAG WHERE USR_SEQ = ?`,
		`DELETE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = ?`,
		`DELETE FROM ALUMNI_VERIFICATION WHERE USR_SEQ = ?`,
		`DELETE credential
		 FROM AUTH_PASSWORD_CREDENTIAL credential
		 JOIN AUTH_IDENTITY identity ON identity.IDENTITY_ID = credential.IDENTITY_ID
		 WHERE identity.ACCOUNT_ID = ?`,
		`UPDATE AUTH_IDENTITY
		 SET SUBJECT_KEY = CONCAT('withdrawn:', ACCOUNT_ID, ':', IDENTITY_ID),
		     NORMALIZED_EMAIL = NULL, STATUS = 'REVOKED', REVOKED_AT = COALESCE(REVOKED_AT, NOW()), UPDATED_AT = NOW()
		 WHERE ACCOUNT_ID = ?`,
		`UPDATE AUTH_ACCOUNT_STATE
		 SET STATUS = 'WITHDRAWN', WITHDRAWN_AT = COALESCE(WITHDRAWN_AT, NOW()), UPDATED_AT = NOW()
		 WHERE ACCOUNT_ID = ?`,
	}
	for index, statement := range statements {
		args := []interface{}{usrSeq}
		if index == 0 {
			args = []interface{}{usrSeq, donorPhone}
		} else if index == 1 || index == 2 || index == 8 {
			args = []interface{}{usrSeq, usrSeq}
		}
		if _, err := tx.Exec(statement, args...); err != nil {
			return nil, fmt.Errorf("account deletion local step %d: %w", index+1, err)
		}
	}
	if phoneClaimsEnabled {
		if _, err := tx.Exec(`DELETE FROM AUTH_PHONE_CLAIM WHERE ACCOUNT_ID = ?`, usrSeq); err != nil {
			return nil, fmt.Errorf("account deletion phone claim step: %w", err)
		}
	}

	for _, provider := range providers {
		if _, err := tx.Exec(`
			INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
				(USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT)
			SELECT ?, ?, ?, 'PENDING', 0, NOW(), NOW(), NOW()
			FROM DUAL
			WHERE NOT EXISTS (
				SELECT 1 FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
				WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = ? AND STATUS IN ('PENDING','PROCESSING')
			)
		`, usrSeq, provider, revocationActionAccountDelete, usrSeq, provider, revocationActionAccountDelete); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = ?`, usrSeq); err != nil {
		return nil, err
	}
	result, err := tx.Exec(`
		UPDATE WEO_MEMBER
		SET USR_ID = CONCAT('withdrawn-', ?), USR_NAME = '탈퇴한 회원',
		    USR_PHONE = '', USR_EMAIL = '', USR_PWD = '', USR_NICK = '', USR_PHOTO = NULL,
		    USR_FN = '', USR_DEPT = '', USR_JOB_CAT = NULL,
		    USR_BIZ_NAME = '', USR_BIZ_DESC = '', USR_BIZ_ADDR = '', USR_POSITION = '',
		    USR_PHONE_PUBLIC = 'N', USR_EMAIL_PUBLIC = 'N', USR_STATUS = 'AAA',
		    USR_ANONYMIZED_AT = COALESCE(USR_ANONYMIZED_AT, NOW()), USR_PURGE_AT = NOW()
		WHERE USR_SEQ = ?
	`, usrSeq, usrSeq)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, errors.New("account deletion did not update exactly one member")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return providers, nil
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

func (r *AuthRepository) RecordAccountDeletionRevocationFailure(usrSeq int, provider string, failure error) error {
	lastError := "provider revocation failed"
	if failure != nil && failure.Error() != "" {
		lastError = failure.Error()
		if len(lastError) > 500 {
			lastError = lastError[:500]
		}
	}
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'PENDING', ATTEMPT_COUNT = ATTEMPT_COUNT + 1,
		    NEXT_ATTEMPT_AT = DATE_ADD(NOW(), INTERVAL 5 MINUTE), LAST_ERROR = ?, UPDATED_AT = NOW()
		WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = ? AND STATUS IN ('PENDING','PROCESSING')
	`, lastError, usrSeq, provider, revocationActionAccountDelete)
	return err
}
