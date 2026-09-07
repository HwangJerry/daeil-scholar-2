// account_erasure_preflight.go — Review retention and provider evidence before external deletion.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"time"
)

func (r *AccountDeletionRequestRepository) PrepareErasure(w model.ErasureWork, validate func(model.DonationRetentionDecision) error) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = verifyDeletionProviders(tx, w.RequestID, w.UserSeq); err != nil {
		return &model.ErasureBlocked{Code: "PROVIDER_REVOCATION_PENDING"}
	}
	s, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	hasDonations := false
	if s["WEO_ORDER"] != nil {
		var conflicting int
		if err = tx.Get(&conflicting, `SELECT COUNT(*) FROM WEO_ORDER WHERE (USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?) AND ((USR_SEQ>0 AND USR_SEQ<>?) OR (O_ACCOUNT_USR_SEQ>0 AND O_ACCOUNT_USR_SEQ<>?))`, w.UserSeq, w.UserSeq, w.UserSeq, w.UserSeq); err != nil {
			return err
		}
		if conflicting > 0 {
			return &model.ErasureBlocked{Code: "DONATION_ACCOUNT_LINK_REVIEW_REQUIRED"}
		}
		var orders []int
		if err = tx.Select(&orders, `SELECT O_SEQ FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?`, w.UserSeq, w.UserSeq); err != nil {
			return err
		}
		hasDonations = len(orders) > 0
		for _, id := range orders {
			var d model.DonationRetentionDecision
			err = tx.Get(&d, `SELECT O_SEQ,BASIS,BASIS_DATE,RETAIN_UNTIL,EVIDENCE_REFERENCE FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=?`, id)
			if err == sql.ErrNoRows && r.DonationRetentionTemplate != nil {
				var donated time.Time
				if err = tx.Get(&donated, `SELECT O_DONATION_DATE FROM WEO_ORDER WHERE O_SEQ=? AND O_SOURCE<>'happy_nanum' AND O_LIFECYCLE_STATUS IN ('completed','partially_refunded','fully_refunded')`, id); err != nil {
					return &model.ErasureBlocked{Code: "DONATION_RETENTION_REVIEW_REQUIRED"}
				}
				d, err = r.DonationRetentionTemplate(id, donated)
				if err == nil {
					err = validate(d)
				}
				if err == nil {
					_, err = tx.Exec(`INSERT INTO ALUMNI_DONATION_RETENTION (O_SEQ,BASIS,BASIS_DATE,RETAIN_UNTIL,EVIDENCE_REFERENCE,REVIEWED_AT) VALUES (?,?,?,?,?,UTC_TIMESTAMP())`, id, d.Basis, retentionSQLDate(d.BasisDate), retentionSQLDate(d.Until), d.Evidence)
				}
			}
			if err != nil {
				return &model.ErasureBlocked{Code: "DONATION_RETENTION_REVIEW_REQUIRED"}
			}
			if err = validate(d); err != nil {
				return &model.ErasureBlocked{Code: "INVALID_DONATION_RETENTION_DECISION"}
			}
		}
	}
	if err = reviewReceiptWork(tx, w, hasDonations); err != nil {
		return err
	}
	return tx.Commit()
}

func retentionSQLDate(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return value.Format("2006-01-02")
}
