// account_erasure_database.go — Atomic application erasure with durable file work.
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"strings"
)

func (r *AccountDeletionRequestRepository) EraseDatabase(w model.ErasureWork, seal func([]byte) ([]byte, error), validate func(model.DonationRetentionDecision) error) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stage string
	if err = tx.Get(&stage, `SELECT STAGE FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? AND MODE='automatic' AND (EXTERNAL_EVIDENCE<>'' OR EXISTS (SELECT 1 FROM ALUMNI_ERASURE_CONTEXT c WHERE c.REQUEST_ID=ALUMNI_ACCOUNT_ERASURE.REQUEST_ID AND c.EXPIRES_AT>UTC_TIMESTAMP())) FOR UPDATE`, w.RequestID); err != nil {
		return err
	}
	if stage == "database_erased" {
		return nil
	}
	var reviewed int
	if err = tx.Get(&reviewed, `SELECT COUNT(*) FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=? AND STATUS<>'unreviewed'`, w.RequestID); err != nil {
		return err
	}
	if reviewed != 1 {
		return &model.ErasureBlocked{Code: "RECEIPT_CONTACT_REVIEW_REQUIRED"}
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
	if err = queueErasureFiles(tx, s, w, r.SiteOrigin); err != nil {
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

// Preserve other authors' conversation by retaining an anonymous, empty parent.
func erasePostChildren(tx *sqlx.Tx, s erasureSchema, user int) error {
	if s["WEO_BOARDBBS"] == nil {
		return nil
	}
	var engine string
	if err := tx.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_BOARDBBS'`); err != nil {
		return err
	}
	if engine != "InnoDB" {
		return &model.ErasureBlocked{Code: "NON_TRANSACTIONAL_TABLE_WEO_BOARDBBS"}
	}
	// Legacy inline replies are not separately owned rows. An operator must
	// separate third-party content before erasure; never silently remove it.
	if s.has("WEO_BOARDBBS", "RE_CONTENTS") {
		var replies int
		if err := tx.Get(&replies, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE USR_SEQ=? AND COALESCE(RE_CONTENTS,'')<>''`, user); err != nil {
			return err
		}
		if replies > 0 {
			return &model.ErasureBlocked{Code: "LEGACY_REPLY_REVIEW_REQUIRED"}
		}
	}
	sets := []string{"USR_SEQ=0"}
	for _, column := range []string{"SUBJECT", "CONTENTS", "CONTENTS_MD", "SUMMARY", "THUMBNAIL_URL", "FILES", "RE_FILES", "USR_NAME", "USR_ID", "EMAIL", "PHONE", "IP", "BBS_IP", "PASSWORD", "REG_ID", "REG_NAME", "REG_EMAIL", "REG_TEL", "REG_PWD", "REG_IPADDR"} {
		if s.has("WEO_BOARDBBS", column) {
			value := "''"
			if column == "SUBJECT" || column == "CONTENTS" {
				value = "'탈퇴한 회원의 삭제된 게시글입니다.'"
			}
			sets = append(sets, "`"+column+"`="+value)
		}
	}
	_, err := tx.Exec("UPDATE WEO_BOARDBBS SET "+strings.Join(sets, ",")+" WHERE USR_SEQ=?", user)
	return err
}
