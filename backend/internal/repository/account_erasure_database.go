// account_erasure_database.go — Atomic application erasure with durable file work.
package repository

import (
	"crypto/sha256"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func (r *AccountDeletionRequestRepository) EraseDatabase(w model.ErasureWork, seal func([]byte) ([]byte, error), validate func(model.DonationRetentionDecision) error) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stage string
	if err = tx.Get(&stage, `SELECT STAGE FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? AND MODE='automatic' AND EXTERNAL_EVIDENCE<>'' FOR UPDATE`, w.RequestID); err != nil {
		return err
	}
	if stage == "database_erased" {
		return nil
	}
	var email string
	if err = tx.Get(&email, `SELECT COALESCE(USR_EMAIL,'') FROM WEO_MEMBER WHERE USR_SEQ=? AND USR_STATUS='AAA' FOR UPDATE`, w.UserSeq); err != nil {
		return err
	}
	if err = verifyDeletionProviders(tx, w.RequestID, w.UserSeq); err != nil {
		return &model.ErasureBlocked{Code: "PROVIDER_REVOCATION_PENDING"}
	}
	s, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	// Subscription tokens require the existing billing-key revocation SOP.
	for _, table := range []string{"SUBSCRIPTION", "WEO_ORDER_PROFILE"} {
		if s[table] != nil {
			var n int
			if err = tx.Get(&n, "SELECT COUNT(*) FROM `"+table+"` WHERE USR_SEQ=?", w.UserSeq); err != nil {
				return err
			}
			if n > 0 {
				return &model.ErasureBlocked{Code: "BILLING_REVOCATION_REVIEW_REQUIRED"}
			}
		}
	}
	if err = queueErasureFiles(tx, s, w); err != nil {
		return err
	}
	if err = eraseDonations(tx, s, w, seal, validate); err != nil {
		return err
	}
	if err = erasePostChildren(tx, s, w.UserSeq); err != nil {
		return err
	}
	if err = eraseAccountReferences(tx, s, w.UserSeq, email); err != nil {
		return err
	}
	if err = s.erase(tx, "WEO_MEMBER", "USR_SEQ=?", w.UserSeq); err != nil {
		return err
	}
	remaining, err := deletionFootprint(tx, w.UserSeq)
	if err != nil {
		return err
	}
	if len(remaining) > 0 {
		return &model.ErasureBlocked{Code: "UNHANDLED_ACCOUNT_REFERENCE"}
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET RETAINED_RECORDS='없음' WHERE REQUEST_ID=? AND RETAINED_RECORDS=''`, w.RequestID); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET STAGE='database_erased',LAST_CODE='',UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, w.RequestID); err != nil {
		return err
	}
	return tx.Commit()
}

func queueErasureFiles(tx *sqlx.Tx, s erasureSchema, w model.ErasureWork) error {
	urls, err := postErasureURLs(tx, s, w.UserSeq)
	if err != nil {
		return err
	}
	for _, col := range []string{"USR_PHOTO", "USR_BIZ_CARD", "USR_THUMNAIL"} {
		if s.has("WEO_MEMBER", col) {
			var url string
			if err := tx.Get(&url, "SELECT COALESCE(`"+col+"`,'') FROM WEO_MEMBER WHERE USR_SEQ=?", w.UserSeq); err != nil {
				return err
			}
			if url != "" {
				urls = append(urls, url)
			}
		}
	}
	var owned []string
	if err := tx.Select(&owned, `SELECT URL_PATH FROM ALUMNI_UPLOAD_OWNER WHERE USR_SEQ=?`, w.UserSeq); err != nil {
		return err
	}
	urls = append(urls, owned...)
	if s["WEO_FILES"] != nil && s["WEO_BOARDBBS"] != nil {
		var attachments []string
		if err := tx.Select(&attachments, `SELECT CONCAT(FILE_PATH,'/',FILE_NAME) FROM WEO_FILES WHERE F_GATE='BB' AND F_JOIN_SEQ IN (SELECT SEQ FROM WEO_BOARDBBS WHERE USR_SEQ=?)`, w.UserSeq); err != nil {
			return err
		}
		urls = append(urls, attachments...)
	}
	for _, url := range urls {
		sum := sha256.Sum256([]byte(url))
		if _, err := tx.Exec(`INSERT IGNORE INTO ALUMNI_ERASURE_FILE (REQUEST_ID,URL_PATH,URL_HASH) VALUES (?,?,?)`, w.RequestID, url, fmt.Sprintf("%x", sum)); err != nil {
			return err
		}
		if s["WEO_FILES"] != nil {
			if err := s.erase(tx, "WEO_FILES", "CONCAT(FILE_PATH,'/',FILE_NAME)=?", url); err != nil {
				return err
			}
		}
	}
	return nil
}

func erasePostChildren(tx *sqlx.Tx, s erasureSchema, user int) error {
	if s["WEO_BOARDBBS"] == nil {
		return nil
	}
	for _, q := range []struct{ table, where string }{
		{"WEO_BOARDLIKE", "BBS_SEQ IN (SELECT SEQ FROM WEO_BOARDBBS WHERE USR_SEQ=?)"},
		{"WEO_BOARDCOMAND", "BC_TYPE='B' AND JOIN_SEQ IN (SELECT SEQ FROM WEO_BOARDBBS WHERE USR_SEQ=?)"},
		{"WEO_BOARDBBS", "USR_SEQ=?"},
	} {
		if err := s.erase(tx, q.table, q.where, user); err != nil {
			return err
		}
	}
	return nil
}
