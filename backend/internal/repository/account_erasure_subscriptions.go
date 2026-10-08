package repository

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"strings"
)

const subscriptionReviewEvidenceMaxBytes = 200
const subscriptionSourceFingerprintLength = 64

// Eligible local references still require external closure proof. Erasure additionally
// requires a root operator's external-closure attestation for this exact source.
func subscriptionErasureReviews(tx *sqlx.Tx, s erasureSchema, id int64, user int) ([]model.ErasureSubscriptionReview, error) {
	out := []model.ErasureSubscriptionReview{}
	if s["SUBSCRIPTION"] == nil {
		return out, nil
	}
	var ids []int
	if err := tx.Select(&ids, `SELECT SUB_SEQ FROM SUBSCRIPTION WHERE USR_SEQ=? ORDER BY SUB_SEQ FOR UPDATE`, user); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return out, nil
	}
	orders := []previewTable{}
	paymentsFinal := true
	if s["WEO_ORDER"] != nil {
		var locked []int
		if err := tx.Select(&locked, `SELECT O_SEQ FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=? FOR UPDATE`, user, user); err != nil {
			return nil, err
		}
		if len(locked) > 0 {
			if s.has("WEO_ORDER", "O_LIFECYCLE_STATUS") {
				var pending int
				if err := tx.Get(&pending, `SELECT COUNT(*) FROM WEO_ORDER WHERE (USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?) AND COALESCE(O_LIFECYCLE_STATUS,'') NOT IN ('completed','fully_refunded','failed','cancelled') FOR UPDATE`, user, user); err != nil {
					return nil, err
				}
				paymentsFinal = pending == 0
			} else {
				paymentsFinal = false
			}
		}
		for _, q := range []previewQuery{
			{table: "WEO_ORDER", action: "billing-source", lockRows: true, where: "USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?", args: []interface{}{user, user}},
			{table: "WEO_PG_DATA", action: "billing-source", lockRows: true, where: "O_SEQ IN (SELECT O_SEQ FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?)", args: []interface{}{user, user}},
		} {
			if s[q.table] != nil {
				table, err := readPreviewTable(tx, q)
				if err != nil {
					return nil, err
				}
				orders = append(orders, table)
			}
		}
	}
	for _, sub := range ids {
		table, err := readPreviewTable(tx, previewQuery{table: "SUBSCRIPTION", action: "billing-source", lockRows: true, where: "SUB_SEQ=? AND USR_SEQ=?", args: []interface{}{sub, user}})
		if err != nil {
			return nil, err
		}
		item := model.ErasureSubscriptionReview{SubscriptionID: sub, SourceFingerprint: previewDigest(append([]previewTable{table}, orders...), nil)}
		if s.has("SUBSCRIPTION", "BILLING_KEY") {
			if err = tx.Get(&item.HasBillingKey, `SELECT COALESCE(BILLING_KEY,'')<>'' FROM SUBSCRIPTION WHERE SUB_SEQ=? FOR UPDATE`, sub); err != nil {
				return nil, err
			}
		}
		if err = tx.Get(&item.Status, `SELECT STATUS FROM SUBSCRIPTION WHERE SUB_SEQ=? FOR UPDATE`, sub); err != nil {
			return nil, err
		}
		linkedPaymentFinal := true
		if s.has("SUBSCRIPTION", "ORDER_SEQ") && s["WEO_ORDER"] != nil {
			var linked int
			if err = tx.Get(&linked, `SELECT COALESCE(ORDER_SEQ,0) FROM SUBSCRIPTION WHERE SUB_SEQ=? FOR UPDATE`, sub); err != nil {
				return nil, err
			}
			if linked > 0 {
				if !s.has("WEO_ORDER", "O_LIFECYCLE_STATUS") {
					linkedPaymentFinal = false
				} else {
					var matched int
					if err = tx.Get(&matched, `SELECT COUNT(*) FROM WEO_ORDER WHERE O_SEQ=? AND (USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?) AND O_LIFECYCLE_STATUS IN ('completed','fully_refunded','failed','cancelled') FOR UPDATE`, linked, user, user); err != nil {
						return nil, err
					}
					linkedPaymentFinal = matched == 1
				}
			}
		}
		item.CanReview = paymentsFinal && linkedPaymentFinal && !item.HasBillingKey && (item.Status == "failed" || item.Status == "cancelled" || item.Status == "pending")
		if s["ALUMNI_ERASURE_SUBSCRIPTION_REVIEW"] != nil {
			var fingerprint string
			err = tx.Get(&fingerprint, `SELECT SOURCE_FINGERPRINT FROM ALUMNI_ERASURE_SUBSCRIPTION_REVIEW WHERE REQUEST_ID=? AND SUB_SEQ=? AND EVIDENCE_REFERENCE<>'' AND OPERATOR_SEQ>0 AND PROVIDER_CLOSURE_OUTCOME IN ('failed','cancelled') FOR UPDATE`, id, sub)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			item.Reviewed = item.CanReview && fingerprint == item.SourceFingerprint
		}
		out = append(out, item)
	}
	return out, nil
}
func billingErasureBlocked(tx *sqlx.Tx, s erasureSchema, id int64, user int) (bool, error) {
	items, err := subscriptionErasureReviews(tx, s, id, user)
	if err != nil {
		return true, err
	}
	for _, item := range items {
		if !item.Reviewed {
			return true, nil
		}
	}
	if s["WEO_ORDER_PROFILE"] != nil {
		var count int
		if err = tx.Get(&count, `SELECT COUNT(*) FROM WEO_ORDER_PROFILE WHERE USR_SEQ=?`, user); err != nil {
			return true, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (r *AccountDeletionRequestRepository) ReviewErasureSubscription(id int64, operator int, request model.AccountDeletionResolution) error {
	evidence := strings.TrimSpace(request.EvidenceReference)
	if operator <= 0 || request.SubscriptionID <= 0 || !request.ExternalClosureConfirmed || (request.ProviderClosureOutcome != "failed" && request.ProviderClosureOutcome != "cancelled") || evidence == "" || len(evidence) > subscriptionReviewEvidenceMaxBytes || len(request.SourceFingerprint) != subscriptionSourceFingerprintLength {
		return &model.ValidationError{Msg: "공급자의 종료 상태(실패·취소), 외부 결제·청구 종료 확인과 200바이트 이내의 근거가 필요합니다."}
	}
	release, locked, err := r.ErasureLock(context.Background())
	if err != nil {
		return err
	}
	if !locked {
		return &model.ValidationError{Msg: "삭제 작업 실행 중입니다. 잠시 후 다시 확인해주세요."}
	}
	defer release()
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var user int
	if err = tx.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND USR_SEQ<>? AND STATUS IN ('pending','processing') FOR UPDATE`, id, operator); err != nil {
		return err
	}
	if r.TestUserSeq > 0 && user != r.TestUserSeq {
		return &model.ValidationError{Msg: "현재 테스트 회원만 처리할 수 있습니다."}
	}
	s, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	items, err := subscriptionErasureReviews(tx, s, id, user)
	if err != nil {
		return err
	}
	found := false
	for _, item := range items {
		if item.SubscriptionID != request.SubscriptionID {
			continue
		}
		found = true
		if subtle.ConstantTimeCompare([]byte(item.SourceFingerprint), []byte(request.SourceFingerprint)) != 1 {
			return ErrErasurePlanChanged
		}
		if !item.CanReview {
			return &model.ErasureBlocked{Code: "BILLING_REVOCATION_REVIEW_REQUIRED"}
		}
		// Retry preserves the original operator/time/evidence; changed source requires a new review.
		if !item.Reviewed {
			if _, err = tx.Exec(`INSERT INTO ALUMNI_ERASURE_SUBSCRIPTION_REVIEW(REQUEST_ID,SUB_SEQ,SOURCE_FINGERPRINT,PROVIDER_CLOSURE_OUTCOME,EVIDENCE_REFERENCE,OPERATOR_SEQ,REVIEWED_AT) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP()) ON DUPLICATE KEY UPDATE SOURCE_FINGERPRINT=VALUES(SOURCE_FINGERPRINT),PROVIDER_CLOSURE_OUTCOME=VALUES(PROVIDER_CLOSURE_OUTCOME),EVIDENCE_REFERENCE=VALUES(EVIDENCE_REFERENCE),OPERATOR_SEQ=VALUES(OPERATOR_SEQ),REVIEWED_AT=VALUES(REVIEWED_AT)`, id, item.SubscriptionID, item.SourceFingerprint, request.ProviderClosureOutcome, evidence, operator); err != nil {
				return err
			}
		}
	}
	if !found {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_ERASURE SET NEXT_ATTEMPT_AT=UTC_TIMESTAMP(),UPDATED_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
