package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthQAErasureOwnershipAndRetention(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	t.Run("new_member_identity_is_not_retired_member_residue", func(t *testing.T) {
		v := service.InternalErasureVerifier{Identifiers: repo}
		result, err := v.EraseTargets(context.Background(), model.ErasureExternalSubject{UserSeq: 99, Phone: defaultGoldenMember.phone, Email: defaultGoldenMember.email, Login: defaultGoldenMember.usrID, RequiredTargets: []string{"other_identifiers"}})
		if err != nil {
			t.Fatal(err)
		}
		if !result[0].Verified() {
			t.Fatal("legitimate new member identity incorrectly holds old erasure")
		}
	})
	t.Run("signup_records_consumed_grant_owner", func(t *testing.T) {
		body := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": goldenPhone}, "", "android", "100", 200)
		id := decodeGolden[model.PhoneVerificationRequestResult](t, body).VerificationID
		body = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": id, "code": goldenCode}, "", "android", "100", 200)
		token := decodeGolden[model.PhoneVerificationConfirmResult](t, body).VerificationToken
		s.request(t, http.MethodPost, "/api/auth/register", map[string]any{"usrId": "ownership_qa", "password": goldenPassword, "name": "합성 회원", "phone": goldenPhone, "email": "ownership@example.test", "fn": "20", "fmDept": "영어", "phoneVerificationToken": token, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}, "", "android", "100", 201)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION v JOIN WEO_MEMBER m ON m.USR_SEQ=v.CONSUMED_USR_SEQ WHERE v.APV_ID=? AND v.CONSUMED_YN='Y' AND v.CONSUMED_AT IS NOT NULL AND v.PHONE=m.USR_PHONE`, id)
	})
	t.Run("preview_explains_legacy_SMS_completion_wait", func(t *testing.T) {
		goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,EXPIRES_AT,REG_DATE) VALUES(REPEAT('e',32),?,REPEAT('f',64),DATE_ADD(NOW(),INTERVAL 5 MINUTE),NOW())`, defaultGoldenMember.phone)
		receipt, err := repo.Create(goldenMemberID, strings.Repeat("c", 64))
		if err != nil {
			t.Fatal(err)
		}
		preview, err := repo.PreviewErasure(receipt.ID)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(preview)
		var decoded map[string]json.RawMessage
		json.Unmarshal(b, &decoded)
		var waits []struct {
			Code  string    `json:"code"`
			Count int       `json:"count"`
			Until time.Time `json:"expectedAt"`
		}
		json.Unmarshal(decoded["completionWaits"], &waits)
		if len(waits) != 1 || waits[0].Count != 1 || !waits[0].Until.After(time.Now().Add(24*time.Hour)) {
			t.Fatalf("missing SMS wait metadata: %s", decoded["completionWaits"])
		}
	})
	t.Run("cleanup_keeps_unexpired_grants_even_if_old", func(t *testing.T) {
		goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,VERIFIED_YN,GRANT_TOKEN_HASH,EXPIRES_AT,GRANT_EXPIRES_AT,REG_DATE) VALUES(REPEAT('d',32),'01000000003',REPEAT('f',64),'Y',REPEAT('d',64),DATE_SUB(NOW(),INTERVAL 1 HOUR),DATE_ADD(NOW(),INTERVAL 30 MINUTE),DATE_SUB(NOW(),INTERVAL 25 HOUR))`)
		if _, err := repository.NewPhoneVerificationRepository(s.db).DeleteExpiredBefore(time.Now().Add(-24 * time.Hour)); err != nil {
			t.Fatal(err)
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=REPEAT('d',32)`)
	})
}

func TestAuthQASubscriptionClosedReferenceReview(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	receipt, err := repo.Create(goldenMemberID, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `INSERT INTO SUBSCRIPTION (SUB_SEQ,USR_SEQ,AMOUNT,PAY_TYPE,STATUS,START_DATE,NEXT_BILL,REG_DATE) VALUES(1,? ,1000,'CARD','failed',NOW(),NOW(),NOW())`, goldenMemberID)
	preview, err := repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(preview)
	var decoded map[string]json.RawMessage
	json.Unmarshal(raw, &decoded)
	var reviews []struct {
		ID          int    `json:"subscriptionId"`
		Fingerprint string `json:"sourceFingerprint"`
		Eligible    bool   `json:"canReview"`
	}
	json.Unmarshal(decoded["subscriptions"], &reviews)
	if len(reviews) != 1 || !reviews[0].Eligible || len(reviews[0].Fingerprint) != 64 {
		t.Fatalf("no official review candidate: %s", decoded["subscriptions"])
	}
	t.Run("unconfirmed_closure_and_cross_account_review_are_rejected", func(t *testing.T) {
		svc := service.AccountDeletionRequestService{Store: repo}
		for _, request := range []model.AccountDeletionResolution{
			{Action: "subscription_review", SubscriptionID: 1, SourceFingerprint: reviews[0].Fingerprint, EvidenceReference: "synthetic-ref"},
			{Action: "subscription_review", SubscriptionID: 1, SourceFingerprint: reviews[0].Fingerprint, ExternalClosureConfirmed: true},
			{Action: "subscription_review", SubscriptionID: 2, SourceFingerprint: reviews[0].Fingerprint, ExternalClosureConfirmed: true, ProviderClosureOutcome: "failed", EvidenceReference: "synthetic-ref"},
		} {
			if err := svc.Resolve(receipt.ID, 999, request); err == nil {
				t.Fatal("incomplete/cross-account closure accepted")
			}
		}
		goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_ERASURE_SUBSCRIPTION_REVIEW`)
	})
	t.Run("pending_active_unknown_or_existing_key_cannot_be_attested_away", func(t *testing.T) {
		for _, test := range []struct{ status, key string }{{"pending", ""}, {"active", ""}, {"failed", "still-active-key"}, {"cancelled", "still-active-key"}, {"unknown", ""}} {
			goldenExec(t, s.db, `UPDATE SUBSCRIPTION SET STATUS=?,BILLING_KEY=? WHERE SUB_SEQ=1`, test.status, test.key)
			fresh, err := repo.PreviewErasure(receipt.ID)
			if err != nil {
				t.Fatal(err)
			}
			item := fresh.Subscriptions[0]
			if (item.CanReview && test.status != "pending") || item.Reviewed {
				t.Fatalf("unsafe reference reviewable: %s", test.status)
			}
			outcome := "cancelled"
			if test.status == "pending" {
				outcome = ""
			}
			request := model.AccountDeletionResolution{Action: "subscription_review", SubscriptionID: 1, SourceFingerprint: item.SourceFingerprint, ExternalClosureConfirmed: true, ProviderClosureOutcome: outcome, EvidenceReference: "synthetic-ref"}
			if err := (&service.AccountDeletionRequestService{Store: repo}).Resolve(receipt.ID, 999, request); err == nil {
				t.Fatal("unsafe reference accepted")
			}
			err = repo.PrepareErasure(model.ErasureWork{RequestID: receipt.ID, UserSeq: goldenMemberID}, service.ValidateDonationRetention)
			var blocked *model.ErasureBlocked
			if !errors.As(err, &blocked) || blocked.Code != "BILLING_REVOCATION_REVIEW_REQUIRED" {
				t.Fatalf("worker must hold unsafe reference: %v", err)
			}
		}
		goldenExec(t, s.db, `UPDATE SUBSCRIPTION SET STATUS='failed',BILLING_KEY=NULL WHERE SUB_SEQ=1`)
	})
	// Source was restored to the original terminal snapshot.
	var resolution model.AccountDeletionResolution
	payload, _ := json.Marshal(map[string]any{"action": "subscription_review", "subscriptionId": 1, "sourceFingerprint": reviews[0].Fingerprint, "externalClosureConfirmed": true, "providerClosureOutcome": "failed", "evidenceReference": "synthetic-provider-terminal-reference"})
	json.Unmarshal(payload, &resolution)
	svc := service.AccountDeletionRequestService{Store: repo}
	if err := svc.Resolve(receipt.ID, 999, resolution); err != nil {
		t.Fatalf("confirmed terminal reference review failed: %v", err)
	}
	if err := svc.Resolve(receipt.ID, 999, resolution); err != nil {
		t.Fatalf("review retry failed: %v", err)
	}
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range preview.Blockers {
		if code == "BILLING_REVOCATION_REVIEW_REQUIRED" {
			t.Fatal("confirmed terminated subscription still blocks worker")
		}
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ERASURE_SUBSCRIPTION_REVIEW WHERE REQUEST_ID=? AND SUB_SEQ=1 AND OPERATOR_SEQ=999`, receipt.ID)
	goldenExec(t, s.db, `UPDATE SUBSCRIPTION SET STATUS='pending' WHERE SUB_SEQ=1`)
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	blocked := false
	for _, code := range preview.Blockers {
		blocked = blocked || code == "BILLING_REVOCATION_REVIEW_REQUIRED"
	}
	if !blocked {
		t.Fatal("changed/pending subscription bypassed hold")
	}
	if err := svc.Resolve(receipt.ID, 999, resolution); err == nil {
		t.Fatal("stale approval accepted")
	}
	goldenExec(t, s.db, `UPDATE SUBSCRIPTION SET STATUS='cancelled' WHERE SUB_SEQ=1`)
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	resolution.SourceFingerprint = preview.Subscriptions[0].SourceFingerprint
	if err := svc.Resolve(receipt.ID, 999, resolution); err != nil {
		t.Fatal(err)
	}
	if err := repo.ControlSchedule(receipt.ID, 999, "expedite"); err != nil {
		t.Fatal(err)
	}
	worker := authQAErasureWorker(t, repo)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM SUBSCRIPTION WHERE USR_SEQ=?`, goldenMemberID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ERASURE_SUBSCRIPTION_REVIEW WHERE REQUEST_ID=?`, receipt.ID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='completed'`, receipt.ID)
}

// Exercise the existing database_erased resume path with actual deletion,
// grant consumption, TTL cleanup and all four target states persisted in MariaDB.
func TestAuthQAErasureWorkerResumeAfterRejoin(t *testing.T) {
	s := newGoldenServer(t)
	oldMember := defaultGoldenMember
	oldMember.phone = goldenPhone
	seedGoldenMemberAs(t, s.db, oldMember)
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,EXPIRES_AT,REG_DATE,CONSUMED_YN,CONSUMED_USR_SEQ,CONSUMED_AT) VALUES(REPEAT('a',32),?,REPEAT('f',64),NOW(),NOW(),'Y',?,NOW()),(REPEAT('b',32),?,REPEAT('f',64),NOW(),NOW(),'N',NULL,NULL)`, oldMember.phone, goldenMemberID, oldMember.phone)
	receipt, err := repo.Create(goldenMemberID, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,EXPIRES_AT,REG_DATE) VALUES(REPEAT('e',32),?,REPEAT('f',64),NOW(),NOW())`, oldMember.phone)
	if err := repo.ExpediteReviewed(receipt.ID, 999, preview.PlanDigest); err == nil {
		t.Fatal("new SMS wait failed to invalidate reviewed plan")
	}
	goldenExec(t, s.db, `DELETE FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=REPEAT('e',32)`)
	goldenExec(t, s.db, `CREATE TABLE QA_ACCOUNT_REFERENCE(USR_SEQ INT,NOTE TEXT) ENGINE=InnoDB`)
	goldenExec(t, s.db, `INSERT INTO QA_ACCOUNT_REFERENCE VALUES(?, 'new unexplained reference')`, goldenMemberID)
	if err := repo.ExpediteReviewed(receipt.ID, 999, preview.PlanDigest); err == nil {
		t.Fatal("new unknown account reference failed to invalidate plan")
	}
	goldenExec(t, s.db, `DROP TABLE QA_ACCOUNT_REFERENCE`)
	worker := authQAErasureWorker(t, repo)
	if err := repo.ControlSchedule(receipt.ID, 999, "expedite"); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=?`, goldenMemberID)
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE CONSUMED_USR_SEQ=?`, goldenMemberID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ERASURE_TARGET WHERE REQUEST_ID=? AND TARGET='other_identifiers' AND LAST_CODE='PHONE_VERIFICATION_RETENTION_PENDING' AND WAIT_COUNT=1 AND WAIT_UNTIL IS NOT NULL`, receipt.ID)
	targets, err := repo.ErasureTargets(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.Name == "other_identifiers" && (target.WaitCount != 1 || target.WaitUntil == nil) {
			t.Fatal("queue dropped SMS wait metadata")
		}
	}
	member := oldMember
	body := s.request(t, http.MethodPost, "/api/auth/phone/verification/request", map[string]string{"phone": member.phone}, "", "android", "100", 200)
	verificationID := decodeGolden[model.PhoneVerificationRequestResult](t, body).VerificationID
	body = s.request(t, http.MethodPost, "/api/auth/phone/verification/confirm", map[string]string{"verificationId": verificationID, "code": goldenCode}, "", "android", "100", 200)
	grantToken := decodeGolden[model.PhoneVerificationConfirmResult](t, body).VerificationToken
	body = s.request(t, http.MethodPost, "/api/auth/register", map[string]any{"usrId": member.usrID, "password": goldenPassword, "name": member.name, "phone": member.phone, "email": member.email, "fn": "20", "fmDept": "영어", "phoneVerificationToken": grantToken, "privacyConsent": map[string]any{"version": goldenConsentVersion, "accepted": true}}, "", "android", "100", 201)
	member.seq = decodeGolden[model.AuthUser](t, body).USRSeq
	if member.seq <= goldenMemberID {
		t.Fatal("rejoin must create a new member")
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=? AND CONSUMED_USR_SEQ=?`, verificationID, member.seq)
	goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,EXPIRES_AT,GRANT_EXPIRES_AT,GRANT_TOKEN_HASH,REG_DATE,VERIFIED_YN) VALUES(REPEAT('d',32),?,REPEAT('f',64),NOW(),DATE_ADD(NOW(),INTERVAL 30 MINUTE),REPEAT('d',64),NOW(),'Y')`, member.phone)
	grant := repository.NewPhoneVerificationRepository(s.db)
	if phone, err := grant.ConsumeGrantForMember(strings.Repeat("d", 64), "01099999999", member.seq); err != nil || phone != "" {
		t.Fatal("mismatched phone accepted")
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE APV_ID=REPEAT('d',32) AND CONSUMED_YN='N'`)
	if phone, err := grant.ConsumeGrantForMember(strings.Repeat("d", 64), member.phone, member.seq); err != nil || phone != member.phone {
		t.Fatalf("new ownership bind: %v", err)
	}
	retry := func() {
		t.Helper()
		goldenExec(t, s.db, `UPDATE ALUMNI_ACCOUNT_ERASURE SET NEXT_ATTEMPT_AT=UTC_TIMESTAMP() WHERE REQUEST_ID=?`, receipt.ID)
		if err := worker.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	retry()
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='processing'`, receipt.ID)
	goldenExec(t, s.db, `UPDATE ALUMNI_PHONE_VERIFICATION SET REG_DATE=DATE_SUB(NOW(),INTERVAL 25 HOUR) WHERE APV_ID=REPEAT('b',32)`)
	goldenExec(t, s.db, `CREATE TRIGGER qa_fail_sms_cleanup BEFORE DELETE ON ALUMNI_PHONE_VERIFICATION FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic cleanup failure'`)
	if _, err := grant.DeleteExpiredBefore(time.Now().Add(-24 * time.Hour)); err == nil {
		t.Fatal("cleanup failure fixture did not fail")
	}
	retry()
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='processing'`, receipt.ID)
	goldenExec(t, s.db, `DROP TRIGGER qa_fail_sms_cleanup`)
	if _, err := grant.DeleteExpiredBefore(time.Now().Add(-24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	// A typed new member's phone is exempt; an unexplained free-text copy is not.
	goldenExec(t, s.db, `CREATE TABLE QA_FREE_TEXT(USR_SEQ INT,NOTE TEXT) ENGINE=InnoDB`)
	goldenExec(t, s.db, `INSERT INTO QA_FREE_TEXT VALUES(?,?)`, member.seq, member.phone)
	retry()
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='processing'`, receipt.ID)
	goldenExec(t, s.db, `DROP TABLE QA_FREE_TEXT`)
	retry()
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='completed' AND USR_SEQ IS NULL`, receipt.ID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=?`, member.seq)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE ACCOUNT_ID=?`, member.seq)
	goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION WHERE CONSUMED_USR_SEQ=?`, member.seq)
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type authQASentryEmpty struct{}

func (authQASentryEmpty) CountUserEvents(context.Context, string) (int, error) { return 0, nil }
func authQAErasureWorker(t *testing.T, repo *repository.AccountDeletionRequestRepository) *service.AutomaticErasureService {
	t.Helper()
	cipher, err := service.NewErasureContextCipher(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rotation.json")
	now := time.Now().UTC()
	data, _ := json.Marshal(map[string]any{"checkedAt": now.Format(time.RFC3339), "retentionDays": 28, "newestBackupAt": now.Format(time.RFC3339), "oldestBackupAgeDays": 1, "journalRetentionDays": 28, "httpLogMaxAgeDays": 28})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return &service.AutomaticErasureService{Store: repo, ContextCipher: cipher, External: &service.InternalErasureVerifier{Identifiers: repo, Sentry: authQASentryEmpty{}, StatusPath: path}, Seal: func(data []byte) ([]byte, error) { return data, nil }}
}

func TestAuthQAPhoneCleanupRetentionBoundary(t *testing.T) {
	s := newGoldenServer(t)
	repo := repository.NewPhoneVerificationRepository(s.db)
	clock := time.Date(2001, 1, 2, 12, 0, 0, 0, time.UTC)
	cutoff := clock.Add(-model.PhoneVerificationRetention)
	for i, issued := range []time.Time{cutoff.Add(-time.Second), cutoff, cutoff.Add(time.Second)} {
		goldenExec(t, s.db, `INSERT INTO ALUMNI_PHONE_VERIFICATION(APV_ID,PHONE,CODE_HASH,EXPIRES_AT,REG_DATE) VALUES(?, '01000000004', REPEAT('f',64),DATE_SUB(NOW(),INTERVAL 1 HOUR),?)`, fmt.Sprintf("%032d", i), issued)
	}
	deleted, err := repo.DeleteExpiredBefore(cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("24h boundary: removed %d records", deleted)
	}
	goldenCount(t, s.db, 2, `SELECT COUNT(*) FROM ALUMNI_PHONE_VERIFICATION`)
	deleted, err = repo.DeleteExpiredBefore(clock.Add(model.PhoneVerificationCleanupInterval - model.PhoneVerificationRetention))
	if err != nil || deleted != 2 {
		t.Fatalf("next hourly cleanup: %d %v", deleted, err)
	}
}

func TestAuthQAErasureProviderSubjectOwnership(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	s.deps.authRepo.EnableCanonicalIdentityWrites()
	const subject = "synthetic-rejoined-provider-subject"
	social := service.NewSocialAuthService(s.deps.authService, service.NewMobileSessionIssuer(s.deps.authService), service.NewSocialLinkTokenStore(s.deps.cacheStore), nil, authQASocialVerifier{subject: subject})
	if _, err := social.LinkIdentity(context.Background(), goldenMemberID, "kakao", model.KakaoAuthorization{AccessToken: "synthetic-provider-token"}); err != nil {
		t.Fatal(err)
	}
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	v := service.InternalErasureVerifier{Identifiers: repo}
	retired := model.ErasureExternalSubject{UserSeq: 99, ProviderSubjects: []string{"KT:" + subject}, RequiredTargets: []string{"other_identifiers"}}
	targets, err := v.EraseTargets(context.Background(), retired)
	if err != nil || !targets[0].Verified() {
		t.Fatalf("new provider identity held retired account: %v", err)
	}
	goldenExec(t, s.db, `UPDATE AUTH_IDENTITY SET STATUS='REVOKED' WHERE PROVIDER='KAKAO' AND ACCOUNT_ID=?`, goldenMemberID)
	targets, err = v.EraseTargets(context.Background(), retired)
	if err != nil || targets[0].Verified() {
		t.Fatal("revoked/uncertain identity was silently exempted")
	}
}

func TestAuthQASubscriptionConfirmedPendingClosure(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	receipt, err := repo.Create(goldenMemberID, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `INSERT INTO SUBSCRIPTION(SUB_SEQ,USR_SEQ,AMOUNT,PAY_TYPE,STATUS,START_DATE,NEXT_BILL,REG_DATE) VALUES(1,?,1000,'CARD','pending',NOW(),NOW(),NOW())`, goldenMemberID)
	preview, err := repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	var request model.AccountDeletionResolution
	payload, _ := json.Marshal(map[string]any{"action": "subscription_review", "subscriptionId": 1, "sourceFingerprint": preview.Subscriptions[0].SourceFingerprint, "externalClosureConfirmed": true, "providerClosureOutcome": "cancelled", "evidenceReference": "synthetic-provider-cancelled-proof"})
	json.Unmarshal(payload, &request)
	if err := (&service.AccountDeletionRequestService{Store: repo}).Resolve(receipt.ID, 999, request); err != nil {
		t.Fatalf("externally confirmed cancelled reference has no official cleanup path: %v", err)
	}
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Subscriptions[0].Reviewed {
		t.Fatal("confirmed pending reference still holds worker")
	}
	if err := repo.ControlSchedule(receipt.ID, 999, "expedite"); err != nil {
		t.Fatal(err)
	}
	if err := authQAErasureWorker(t, repo).RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM SUBSCRIPTION WHERE USR_SEQ=?`, goldenMemberID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS='completed'`, receipt.ID)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_ERASURE_SUBSCRIPTION_REVIEW WHERE REQUEST_ID=? AND PROVIDER_CLOSURE_OUTCOME='cancelled'`, receipt.ID)
}

func TestAuthQASubscriptionPaymentAndPGChangesKeepHold(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	donations := repository.NewDonateRepository(s.db)
	order, err := donations.InsertOrder(goldenMemberID, "EP", "CARD", 1000, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	repo := &repository.AccountDeletionRequestRepository{DB: s.db, WaitHours: 24}
	receipt, err := repo.Create(goldenMemberID, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, s.db, `INSERT INTO SUBSCRIPTION(SUB_SEQ,USR_SEQ,AMOUNT,PAY_TYPE,STATUS,ORDER_SEQ,START_DATE,NEXT_BILL,REG_DATE) VALUES(1,?,1000,'CARD','failed',?,NOW(),NOW(),NOW())`, goldenMemberID, order)
	preview, err := repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Subscriptions[0].CanReview {
		t.Fatal("unconfirmed payment marked reviewable")
	}
	request := model.AccountDeletionResolution{Action: "subscription_review", SubscriptionID: 1, SourceFingerprint: preview.Subscriptions[0].SourceFingerprint, ExternalClosureConfirmed: true, ProviderClosureOutcome: "failed", EvidenceReference: "synthetic-provider-terminal"}
	svc := service.AccountDeletionRequestService{Store: repo}
	if err := svc.Resolve(receipt.ID, 999, request); err == nil {
		t.Fatal("attestation bypassed unfinished payment")
	}
	goldenExec(t, s.db, `UPDATE WEO_ORDER SET O_LIFECYCLE_STATUS='completed',O_PAYMENT='Y',O_PAY=1000,O_NET_RECEIVED_AMOUNT=1000 WHERE O_SEQ=?`, order)
	if err := svc.Resolve(receipt.ID, 999, request); !errors.Is(err, repository.ErrErasurePlanChanged) {
		t.Fatalf("changed payment accepted old review: %v", err)
	}
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	request.SourceFingerprint = preview.Subscriptions[0].SourceFingerprint
	if err := svc.Resolve(receipt.ID, 999, request); err != nil {
		t.Fatal(err)
	}
	if _, err := donations.InsertPGData(&model.PGData{CNO: "synthetic-transaction", ResCD: "0000", Amount: 1000, OSeq: int(order)}); err != nil {
		t.Fatal(err)
	}
	preview, err = repo.PreviewErasure(receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Subscriptions[0].Reviewed {
		t.Fatal("new PG source kept prior approval")
	}
	err = repo.PrepareErasure(model.ErasureWork{RequestID: receipt.ID, UserSeq: goldenMemberID}, service.ValidateDonationRetention)
	var blocked *model.ErasureBlocked
	if !errors.As(err, &blocked) || blocked.Code != "BILLING_REVOCATION_REVIEW_REQUIRED" {
		t.Fatalf("worker failed to hold changed payment source: %v", err)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_ORDER WHERE O_SEQ=?`, order)
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_PG_DATA WHERE O_SEQ=?`, order)
}
