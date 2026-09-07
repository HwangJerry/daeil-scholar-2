// account_erasure_context.go — Bounded protected handoff independent of member rows.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
)

// Ten days matches the existing operator response window. Expiry requires
// manual review, never false completion or indefinite identifier retention.
const erasureContextLifetimeDays = 10
const maxErasureContextBytes = 1 << 20

func (r *AccountDeletionRequestRepository) SaveErasureContext(id int64, encrypted []byte) error {
	if len(encrypted) == 0 || len(encrypted) > maxErasureContextBytes {
		return &model.ErasureBlocked{Code: "ERASURE_CONTEXT_UNREADABLE"}
	}
	result, err := r.DB.Exec(`INSERT INTO ALUMNI_ERASURE_CONTEXT (REQUEST_ID,CIPHERTEXT,EXPIRES_AT,CREATED_AT)
 SELECT e.REQUEST_ID,?,DATE_ADD(UTC_TIMESTAMP(),INTERVAL ? DAY),UTC_TIMESTAMP()
 FROM ALUMNI_ACCOUNT_ERASURE e JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=e.REQUEST_ID
 WHERE e.REQUEST_ID=? AND e.MODE='automatic' AND e.STAGE<>'database_erased' AND d.STATUS<>'completed'
 ON DUPLICATE KEY UPDATE REQUEST_ID=VALUES(REQUEST_ID)`, encrypted, erasureContextLifetimeDays, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Existing context is immutable, including its original expiry.
		_, err = r.LoadErasureContext(id)
		if err == sql.ErrNoRows {
			return &model.ErasureBlocked{Code: "ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED"}
		}
		return err
	}
	return nil
}
func (r *AccountDeletionRequestRepository) LoadErasureContext(id int64) ([]byte, error) {
	var encrypted []byte
	err := r.DB.Get(&encrypted, `SELECT CIPHERTEXT FROM ALUMNI_ERASURE_CONTEXT WHERE REQUEST_ID=? AND EXPIRES_AT>UTC_TIMESTAMP()`, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return encrypted, err
}

// Refresh only a live pre-erasure handoff; identifiers keep their original expiry.
func (r *AccountDeletionRequestRepository) RefreshErasureContext(id int64, encrypted []byte) error {
	if len(encrypted) == 0 || len(encrypted) > maxErasureContextBytes {
		return &model.ErasureBlocked{Code: "ERASURE_CONTEXT_UNREADABLE"}
	}
	result, err := r.DB.Exec(`UPDATE ALUMNI_ERASURE_CONTEXT c JOIN ALUMNI_ACCOUNT_ERASURE e ON e.REQUEST_ID=c.REQUEST_ID
 SET c.CIPHERTEXT=? WHERE c.REQUEST_ID=? AND c.EXPIRES_AT>UTC_TIMESTAMP() AND e.MODE='automatic' AND e.STAGE<>'database_erased' AND e.STAGE<>'completed'`, encrypted, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return &model.ErasureBlocked{Code: "ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED"}
	}
	return nil
}
