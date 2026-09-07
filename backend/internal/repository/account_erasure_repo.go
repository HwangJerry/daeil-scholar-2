// account_erasure_repo.go — Durable scheduling, operator takeover and completion.
package repository

import (
	"context"
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
)

func (r *AccountDeletionRequestRepository) ErasureBatch(ctx context.Context) ([]model.ErasureWork, error) {
	rows := []model.ErasureWork{}
	err := r.DB.SelectContext(ctx, &rows, `SELECT d.REQUEST_ID,d.USR_SEQ,e.STAGE,e.EXTERNAL_EVIDENCE
        FROM ALUMNI_ACCOUNT_DELETION_REQUEST d JOIN ALUMNI_ACCOUNT_ERASURE e ON e.REQUEST_ID=d.REQUEST_ID
        WHERE e.MODE='automatic' AND d.STATUS <> 'completed' AND e.NEXT_ATTEMPT_AT<=UTC_TIMESTAMP()
        AND (?=0 OR d.USR_SEQ=?) ORDER BY e.NEXT_ATTEMPT_AT,d.REQUEST_ID LIMIT 10`, r.TestUserSeq, r.TestUserSeq)
	return rows, err
}

// One server owns the batch. The named lock lives on a dedicated connection.
func (r *AccountDeletionRequestRepository) ErasureLock(ctx context.Context) (func(), bool, error) {
	conn, err := r.DB.Connx(ctx)
	if err != nil {
		return nil, false, err
	}
	var locked int
	if err = conn.GetContext(ctx, &locked, `SELECT GET_LOCK('dflh_account_erasure',0)`); err != nil || locked != 1 {
		conn.Close()
		return nil, false, err
	}
	return func() {
		conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK('dflh_account_erasure')`)
		conn.Close()
	}, true, nil
}

func (r *AccountDeletionRequestRepository) SetErasureMode(id int64, operator int, mode string) error {
	if mode != "automatic" && mode != "manual" {
		return &model.ValidationError{Msg: "처리 방식을 확인해주세요."}
	}
	result, err := r.DB.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE e JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=e.REQUEST_ID
        SET e.MODE=?,e.LAST_CODE='',e.NEXT_ATTEMPT_AT=UTC_TIMESTAMP(),e.UPDATED_AT=UTC_TIMESTAMP()
        WHERE d.REQUEST_ID=? AND d.STATUS<>'completed' AND d.USR_SEQ<>?`, mode, id, operator)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AccountDeletionRequestRepository) ErasureActive(id int64) (bool, error) {
	var n int
	err := r.DB.Get(&n, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_ERASURE e JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=e.REQUEST_ID
        WHERE e.REQUEST_ID=? AND e.MODE='automatic' AND d.STATUS<>'completed'`, id)
	return n == 1, err
}

func (r *AccountDeletionRequestRepository) ErasureRetry(id int64, code string) error {
	_, err := r.DB.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET LAST_CODE=?,NEXT_ATTEMPT_AT=DATE_ADD(UTC_TIMESTAMP(),INTERVAL 5 MINUTE),UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, code, id)
	return err
}

func (r *AccountDeletionRequestRepository) ErasureExternalSubject(w model.ErasureWork) (model.ErasureExternalSubject, error) {
	s := model.ErasureExternalSubject{RequestID: w.RequestID, UserSeq: w.UserSeq, ProviderSubjects: []string{}}
	err := r.DB.QueryRow(`SELECT COALESCE(USR_ID,''),COALESCE(USR_EMAIL,''),COALESCE(USR_PHONE,'') FROM WEO_MEMBER WHERE USR_SEQ=?`, w.UserSeq).Scan(&s.Login, &s.Email, &s.Phone)
	if err != nil {
		return s, err
	}
	err = r.DB.Select(&s.ProviderSubjects, `SELECT CONCAT(NMS_GATE,':',NMS_ID) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=?`, w.UserSeq)
	if err != nil {
		return s, err
	}
	var hasOrders int
	if err = r.DB.Get(&hasOrders, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_ORDER'`); err != nil {
		return s, err
	}
	s.Retentions = []model.DonationRetentionDecision{}
	if hasOrders > 0 {
		err = r.DB.Select(&s.Retentions, `SELECT d.O_SEQ,d.BASIS,d.BASIS_DATE,d.RETAIN_UNTIL,d.EVIDENCE_REFERENCE FROM ALUMNI_DONATION_RETENTION d JOIN WEO_ORDER o ON o.O_SEQ=d.O_SEQ WHERE o.USR_SEQ=? OR o.O_ACCOUNT_USR_SEQ=?`, w.UserSeq, w.UserSeq)
	}
	if err != nil {
		return s, err
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return s, err
	}
	defer tx.Rollback()
	schema, err := readErasureSchema(tx)
	if err != nil {
		return s, err
	}
	urls, _, err := erasureFileCandidates(tx, schema, w)
	if err != nil {
		return s, err
	}
	seen := map[string]bool{}
	for _, raw := range urls {
		_, external, err := model.ErasureFilePath(raw, r.SiteOrigin)
		if err != nil {
			return s, err
		}
		if external && !seen[raw] {
			s.ExternalFileURLs = append(s.ExternalFileURLs, raw)
			seen[raw] = true
		}
	}
	return s, nil
}

func (r *AccountDeletionRequestRepository) RecordExternalErasure(id int64, evidence string) error {
	if strings.TrimSpace(evidence) == "" || len(evidence) > 200 {
		return &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE e JOIN ALUMNI_ACCOUNT_DELETION_REQUEST d ON d.REQUEST_ID=e.REQUEST_ID
 SET e.EXTERNAL_EVIDENCE=?,e.UPDATED_AT=UTC_TIMESTAMP() WHERE e.REQUEST_ID=? AND e.MODE='automatic' AND d.STATUS<>'completed'`, evidence, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_TARGET SET STATUS='complete',EVIDENCE_REFERENCE=?,LAST_CODE='',UPDATED_AT=UTC_TIMESTAMP()
 WHERE REQUEST_ID=? AND STATUS NOT IN ('complete','not_applicable')`, evidence, id); err != nil {
		return err
	}
	verified, e := verifiedErasureTargets(tx, id)
	if e != nil {
		return e
	}
	if !verified {
		return ErrDeletionIncomplete
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_ERASURE_CONTEXT WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AccountDeletionRequestRepository) ErasureFiles(id int64) ([]model.ErasureFile, error) {
	files := []model.ErasureFile{}
	err := r.DB.Select(&files, `SELECT ID,URL_PATH FROM ALUMNI_ERASURE_FILE WHERE REQUEST_ID=? ORDER BY ID LIMIT 100`, id)
	return files, err
}
func (r *AccountDeletionRequestRepository) ErasureFileDone(id int64) error {
	_, err := r.DB.Exec(`DELETE FROM ALUMNI_ERASURE_FILE WHERE ID=?`, id)
	return err
}

func (r *AccountDeletionRequestRepository) FinishAutomaticErasure(w model.ErasureWork) error {
	active, err := r.ErasureActive(w.RequestID)
	if err != nil {
		return err
	}
	if !active {
		return sql.ErrNoRows
	}
	var ready int
	err = r.DB.Get(&ready, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_ERASURE e WHERE REQUEST_ID=? AND STAGE='database_erased'
        AND EXTERNAL_EVIDENCE<>'' AND NOT EXISTS (SELECT 1 FROM ALUMNI_ERASURE_FILE f WHERE f.REQUEST_ID=e.REQUEST_ID)`, w.RequestID)
	if err != nil {
		return err
	}
	if ready != 1 {
		return ErrDeletionIncomplete
	}
	var result model.AccountDeletionReceipt
	if err = r.DB.Get(&result, `SELECT `+deletionReceiptColumns+` FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=?`, w.RequestID); err != nil {
		return err
	}
	resolution := model.AccountDeletionResolution{RetainedRecords: result.RetainedRecords, EvidenceReference: "automatic: verified database, files, provider revocation and external processor; receipt confirmation"}
	if result.RetentionUntil != nil {
		resolution.RetentionUntil = result.RetentionUntil.Format("2006-01-02")
	}
	// The existing private receipt is the durable in-app completion confirmation.
	return r.Complete(w.RequestID, 0, resolution)
}
