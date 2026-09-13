// account_erasure_preview_checks.go — Read-only conditions that would hold the erasure worker.
package repository

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// previewBlockers reports, without writing, the worker's own hold conditions
// so an operator sees them before approving. Codes match ErasureBlocked codes.
func previewBlockers(tx *sqlx.Tx, s erasureSchema, id int64, user int) ([]string, error) {
	blockers := []string{}
	if s.has("WEO_MEMBER", "USR_STATUS") {
		var status string
		err := tx.Get(&status, `SELECT COALESCE(USR_STATUS,'') FROM WEO_MEMBER WHERE USR_SEQ=?`, user)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil && status != "AAA" {
			blockers = append(blockers, "MEMBER_NOT_WITHDRAWN")
		}
	}
	for _, table := range []string{"SUBSCRIPTION", "WEO_ORDER_PROFILE"} {
		if s[table] != nil {
			var n int
			if err := tx.Get(&n, "SELECT COUNT(*) FROM `"+table+"` WHERE USR_SEQ=?", user); err != nil {
				return nil, err
			}
			if n > 0 {
				blockers = append(blockers, "BILLING_REVOCATION_REVIEW_REQUIRED")
			}
		}
	}
	if s.has("WEO_BOARDBBS", "RE_CONTENTS") {
		var replies int
		if err := tx.Get(&replies, `SELECT COUNT(*) FROM WEO_BOARDBBS WHERE USR_SEQ=? AND COALESCE(RE_CONTENTS,'')<>''`, user); err != nil {
			return nil, err
		}
		if replies > 0 {
			blockers = append(blockers, "LEGACY_REPLY_REVIEW_REQUIRED")
		}
	}
	donationBlockers, hasDonations, err := previewDonationBlockers(tx, s, user)
	if err != nil {
		return nil, err
	}
	blockers = append(blockers, donationBlockers...)
	// Unreviewed receipt work only holds the worker when app donations exist.
	var receipt string
	err = tx.Get(&receipt, `SELECT STATUS FROM ALUMNI_ERASURE_RECEIPT_WORK WHERE REQUEST_ID=?`, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if errors.Is(err, sql.ErrNoRows) || (receipt == "unreviewed" && hasDonations) {
		blockers = append(blockers, "RECEIPT_CONTACT_REVIEW_REQUIRED")
	}
	return blockers, nil
}

func previewDonationBlockers(tx *sqlx.Tx, s erasureSchema, user int) ([]string, bool, error) {
	if s["WEO_ORDER"] == nil {
		return nil, false, nil
	}
	blockers := []string{}
	var conflicting int
	if err := tx.Get(&conflicting, `SELECT COUNT(*) FROM WEO_ORDER WHERE (USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?) AND ((USR_SEQ>0 AND USR_SEQ<>?) OR (O_ACCOUNT_USR_SEQ>0 AND O_ACCOUNT_USR_SEQ<>?))`, user, user, user, user); err != nil {
		return nil, false, err
	}
	if conflicting > 0 {
		blockers = append(blockers, "DONATION_ACCOUNT_LINK_REVIEW_REQUIRED")
	}
	var orders []int
	if err := tx.Select(&orders, `SELECT O_SEQ FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?`, user, user); err != nil {
		return nil, false, err
	}
	for _, order := range orders {
		current, err := donationSourceFingerprint(tx, order)
		if err != nil {
			return nil, false, err
		}
		var reviewed string
		err = tx.Get(&reviewed, `SELECT COALESCE(SOURCE_FINGERPRINT,'') FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=?`, order)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, false, err
		}
		if err != nil || reviewed != current {
			blockers = append(blockers, "DONATION_RETENTION_REVIEW_REQUIRED")
		}
	}
	return blockers, len(orders) > 0, nil
}

// previewEngineBlocker mirrors erasureSchema.erase: a non-InnoDB target cannot
// be rolled back, so the worker refuses it.
func previewEngineBlocker(tx *sqlx.Tx, table string) (string, error) {
	var engine string
	if err := tx.Get(&engine, `SELECT COALESCE(ENGINE,'') FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table); err != nil {
		return "", err
	}
	if engine != "InnoDB" {
		return "NON_TRANSACTIONAL_TABLE_" + table, nil
	}
	return "", nil
}

// annotateDonationRows states, per sampled order, whether minimal fields are
// archived under a reviewed legal basis before the app copy is deleted.
func annotateDonationRows(tx *sqlx.Tx, s erasureSchema, table *previewTable) error {
	column := -1
	for i, name := range table.Columns {
		if name == "O_SEQ" {
			column = i
		}
	}
	if column < 0 || s["ALUMNI_DONATION_RETENTION"] == nil {
		return nil
	}
	for i, row := range table.Rows {
		if row.Before[column] == nil {
			continue
		}
		order, err := strconv.Atoi(*row.Before[column])
		if err != nil {
			continue
		}
		var decision model.DonationRetentionDecision
		err = tx.Get(&decision, `SELECT O_SEQ,BASIS,BASIS_DATE,RETAIN_UNTIL,EVIDENCE_REFERENCE,SOURCE_FINGERPRINT FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=?`, order)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			table.Rows[i].Note = "보존 결정 없음 · 결정이 기록될 때까지 처리 보류"
		case err != nil:
			return err
		case decision.Until != nil && decision.Until.After(time.Now()):
			table.Rows[i].Note = "기부자명·기부일·금액·거래번호만 암호화 보관 후 삭제 · " + decision.Basis + " · " + decision.Until.Format("2006-01-02") + "까지"
		default:
			table.Rows[i].Note = "보존 대상 아님 · 삭제 (" + decision.Basis + ")"
		}
	}
	return nil
}
