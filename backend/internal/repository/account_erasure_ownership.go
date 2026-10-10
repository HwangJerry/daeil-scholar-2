package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

// Only explicitly typed identifier columns may belong to a legitimate different
// member. Free text and orphaned/unknown ownership continue to block erasure.
func identifierOwnerExclusion(s erasureSchema, table, column string, user int) (string, []interface{}) {
	if user <= 0 || !s.has("WEO_MEMBER", "USR_STATUS") {
		return "", nil
	}
	owner := ""
	extra := ""
	switch table {
	case "WEO_MEMBER":
		if column == "USR_ID" || column == "USR_EMAIL" || column == "USR_PHONE" {
			owner = "USR_SEQ"
		}
	case "AUTH_IDENTITY":
		if s.has(table, "STATUS") && (column == "SUBJECT_KEY" || column == "NORMALIZED_EMAIL") {
			owner = "ACCOUNT_ID"
			extra = " AND owned.STATUS='ACTIVE'"
		}
	case "AUTH_PHONE_CLAIM":
		if column == "CANONICAL_PHONE" {
			owner = "ACCOUNT_ID"
		}
	case "WEO_MEMBER_SOCIAL":
		if column == "NMS_ID" || column == "NMS_EMAIL" {
			owner = "USR_SEQ"
		}
	case "ALUMNI_PHONE_VERIFICATION":
		if column == "PHONE" && s.has(table, "CONSUMED_USR_SEQ") {
			owner = "CONSUMED_USR_SEQ"
			extra = " AND owned.CONSUMED_YN='Y' AND owned.CONSUMED_AT IS NOT NULL AND m.USR_PHONE=owned.PHONE"
		}
	}
	if owner == "" || !s.has(table, owner) {
		return "", nil
	}
	// Alias owned is the outer scanned table, m is the current legitimate member.
	return "NOT EXISTS (SELECT 1 FROM WEO_MEMBER m WHERE m.USR_SEQ=owned.`" + owner + "` AND m.USR_SEQ<>? AND m.USR_STATUS<>'AAA'" + extra + ")", []interface{}{user}
}

func smsWaitQuery(s erasureSchema, subject model.ErasureExternalSubject) (previewQuery, bool) {
	phone := model.NormalizePhoneNumber(subject.Phone)
	if !phone.Valid() || s["ALUMNI_PHONE_VERIFICATION"] == nil {
		return previewQuery{}, false
	}
	q := previewQuery{table: "ALUMNI_PHONE_VERIFICATION", action: "wait", where: "REPLACE(REPLACE(PHONE,'-',''),' ','')=?", args: []interface{}{phone.String()}}
	if s.has(q.table, "CONSUMED_USR_SEQ") {
		// Rows that this worker will delete are not a completion wait.
		q.where += " AND NOT (COALESCE(CONSUMED_USR_SEQ,0)=? AND CONSUMED_YN='Y' AND CONSUMED_AT IS NOT NULL)"
		q.args = append(q.args, subject.UserSeq)
		excluded, args := identifierOwnerExclusion(s, q.table, "PHONE", subject.UserSeq)
		// readPreviewTable has no table alias; use the table name in the shared predicate.
		if excluded != "" {
			q.where += " AND " + strings.ReplaceAll(excluded, "owned.", "ALUMNI_PHONE_VERIFICATION.")
			q.args = append(q.args, args...)
		}
	}
	return q, true
}

func smsCompletionWait(tx *sqlx.Tx, s erasureSchema, subject model.ErasureExternalSubject, now time.Time) (*model.ErasureCompletionWait, previewTable, error) {
	q, ok := smsWaitQuery(s, subject)
	if !ok {
		return nil, previewTable{}, nil
	}
	table, err := readPreviewTable(tx, q)
	if err != nil || table.Count == 0 {
		return nil, table, err
	}
	// DB-local timestamps (legacy NOW()) are compared in the DB's timezone.
	var seconds int64
	err = tx.Get(&seconds, `SELECT COALESCE(MAX(TIMESTAMPDIFF(SECOND,NOW(),GREATEST(DATE_ADD(REG_DATE,INTERVAL ? SECOND),CASE WHEN CONSUMED_YN='Y' THEN REG_DATE ELSE GREATEST(EXPIRES_AT,COALESCE(GRANT_EXPIRES_AT,REG_DATE)) END))),0) FROM ALUMNI_PHONE_VERIFICATION WHERE `+q.where, append([]interface{}{int(model.PhoneVerificationRetention.Seconds())}, q.args...)...)
	if err != nil {
		return nil, table, err
	}
	if seconds < 0 {
		seconds = 0
	}
	until := now.UTC().Add(time.Duration(seconds)*time.Second + model.PhoneVerificationCleanupInterval + model.ErasureRecheckInterval + model.ErasurePollInterval)
	return &model.ErasureCompletionWait{Code: model.PhoneVerificationWaitCode, Count: table.Count, ExpectedAt: until}, table, nil
}

func (r *AccountDeletionRequestRepository) SMSCompletionWait(subject model.ErasureExternalSubject) (*model.ErasureCompletionWait, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	s, err := readErasureSchema(tx)
	if err != nil {
		return nil, err
	}
	wait, _, err := smsCompletionWait(tx, s, subject, time.Now())
	return wait, err
}
