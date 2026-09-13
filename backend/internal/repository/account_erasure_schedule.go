package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// Both controls only enqueue work. The same locked worker performs all erasure.
// The global worker lock also serializes operator changes with a running batch.
func (r *AccountDeletionRequestRepository) ControlSchedule(id int64, operator int, action string) error {
	return r.controlSchedule(id, operator, action, nil)
}

// review, when set, runs inside the locked transaction before any change.
func (r *AccountDeletionRequestRepository) controlSchedule(id int64, operator int, action string, review func(*sqlx.Tx, int) error) error {
	if operator <= 0 {
		return &model.ValidationError{Msg: "관리자 확인이 필요합니다."}
	}
	if action != "expedite" && action != "schedule" && action != "retry_social" {
		return &model.ValidationError{Msg: "처리 방식을 확인해주세요."}
	}
	if r.WaitHours <= 0 || r.WaitHours > 24*30 {
		return &model.ValidationError{Msg: "탈퇴 대기 기간 설정이 필요합니다."}
	}
	release, locked, err := r.ErasureLock(context.Background())
	if err != nil {
		return err
	}
	if !locked {
		return &model.ValidationError{Msg: "삭제 작업이 실행 중입니다. 잠시 후 다시 확인해주세요."}
	}
	defer release()
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var user int
	if err = tx.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS IN ('pending','processing') AND USR_SEQ<>? FOR UPDATE`, id, operator); err != nil {
		return err
	}
	if r.TestUserSeq > 0 && user != r.TestUserSeq {
		return &model.ValidationError{Msg: "현재 테스트 회원만 처리할 수 있습니다."}
	}
	if review != nil {
		if err = review(tx, user); err != nil {
			return err
		}
	}
	if action == "retry_social" {
		// Preserve REVOKED evidence: never repeat a completed provider call.
		result, e := tx.Exec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS=CASE WHEN STATUS='FINALIZE_FAILED' THEN 'REVOKED' ELSE 'PENDING' END,CLAIM_TOKEN=NULL,ATTEMPT_COUNT=0,NEXT_ATTEMPT_AT=NOW(),LAST_ERROR=NULL,UPDATED_AT=NOW()
   WHERE USR_SEQ=? AND ACTION='ACCOUNT_DELETE' AND STATUS IN ('FAILED','FINALIZE_FAILED')`, user)
		if e != nil {
			return e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return e
		}
		if n == 0 {
			return sql.ErrNoRows
		}
	} else {
		if _, err = tx.Exec(`INSERT IGNORE INTO ALUMNI_ERASURE_SCHEDULE (REQUEST_ID,SCHEDULED_AT,CREATED_AT) VALUES (?,DATE_ADD(UTC_TIMESTAMP(),INTERVAL ? HOUR),UTC_TIMESTAMP())`, id, r.WaitHours); err != nil {
			return err
		}
		if action == "expedite" {
			if _, err = tx.Exec(`UPDATE ALUMNI_ERASURE_SCHEDULE SET EXPEDITED_AT=COALESCE(EXPEDITED_AT,UTC_TIMESTAMP()),EXPEDITED_BY=COALESCE(EXPEDITED_BY,?) WHERE REQUEST_ID=?`, operator, id); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST d JOIN ALUMNI_ERASURE_SCHEDULE s ON s.REQUEST_ID=d.REQUEST_ID SET d.TARGET_AT=s.SCHEDULED_AT,d.DUE_AT=DATE_ADD(s.SCHEDULED_AT,INTERVAL 10 DAY) WHERE d.REQUEST_ID=?`, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET MODE='automatic',LAST_CODE='',NEXT_ATTEMPT_AT=UTC_TIMESTAMP(),UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Evidence is operator-attested and monotonic. Completion is still checked by the worker.
func (r *AccountDeletionRequestRepository) ReviewErasureTarget(id int64, operator int, target model.ErasureTarget) error {
	valid := false
	for _, name := range model.ErasureTargetNames {
		if name == target.Name {
			valid = true
		}
	}
	if !valid || operator <= 0 || len(target.Evidence) == 0 || len(target.Evidence) > 200 || (target.Status != "manual" && target.Status != "complete" && target.Status != "not_applicable") {
		return &model.ValidationError{Msg: "대상, 처리 상태와 200바이트 이내의 확인 근거가 필요합니다."}
	}
	release, locked, err := r.ErasureLock(context.Background())
	if err != nil {
		return err
	}
	if !locked {
		return &model.ValidationError{Msg: "삭제 작업이 실행 중입니다. 잠시 후 다시 확인해주세요."}
	}
	defer release()
	var user int
	if err = r.DB.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND USR_SEQ<>? AND STATUS IN ('pending','processing')`, id, operator); err != nil {
		return err
	}
	if r.TestUserSeq > 0 && user != r.TestUserSeq {
		return &model.ValidationError{Msg: "현재 테스트 회원만 처리할 수 있습니다."}
	}
	target.Evidence = fmt.Sprintf("admin:%d %s", operator, target.Evidence)
	if len(target.Evidence) > 200 {
		return &model.ValidationError{Msg: "확인 근거를 더 짧게 입력해주세요."}
	}
	if err = r.RecordErasureTargets(id, []model.ErasureTarget{target}); err != nil {
		return err
	}
	_, err = r.DB.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET LAST_CODE='',NEXT_ATTEMPT_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, id)
	return err
}

func (r *AccountDeletionRequestRepository) checkOperatorErasureScope(id int64, operator int) error {
	var user int
	if err := r.DB.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS IN ('pending','processing') AND USR_SEQ<>?`, id, operator); err != nil {
		return err
	}
	if r.TestUserSeq > 0 && user != r.TestUserSeq {
		return &model.ValidationError{Msg: "현재 테스트 회원만 처리할 수 있습니다."}
	}
	return nil
}
