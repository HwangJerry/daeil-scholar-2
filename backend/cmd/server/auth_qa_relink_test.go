// Regression: a freshly verified provider can be relinked to its original member.
package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
)

func TestAuthQASocialRelinkAfterDisconnect(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	s.deps.authRepo.EnableCanonicalIdentityWrites()
	social := service.NewSocialAuthService(s.deps.authService,
		service.NewMobileSessionIssuer(s.deps.authService),
		service.NewSocialLinkTokenStore(s.deps.cacheStore), nil,
		authQASocialVerifier{subject: "synthetic-auth-qa-relink-subject"})
	ctx := context.Background()
	authorization := model.KakaoAuthorization{AccessToken: "synthetic-provider-token"}
	linked, err := social.LinkIdentity(ctx, goldenMemberID, "kakao", authorization)
	if err != nil || len(linked.Providers) != 1 || !linked.HasPassword {
		t.Fatalf("initial link failed: %v", err)
	}
	disconnected, err := s.deps.authService.Disconnect(goldenMemberID, "kakao")
	if err != nil || len(disconnected.Connections.Providers) != 0 || !disconnected.Connections.HasPassword {
		t.Fatalf("disconnect must retain password access: %v", err)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=? AND PROVIDER='KAKAO' AND STATUS='REVOKED'`, goldenMemberID)
	s.login(t, "android") // Control: the native login still works.
	_, err = social.LinkIdentity(ctx, goldenMemberID, "kakao", authorization)
	if err != nil {
		t.Fatalf("verified provider must be relinkable after disconnect: %v", err)
	}
	goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=? AND PROVIDER='KAKAO' AND STATUS='ACTIVE'`, goldenMemberID)
}

func TestClaimedDisconnectCannotTouchReplacementCredential(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	s.deps.authRepo.EnableCanonicalIdentityWrites()
	repo := s.deps.authRepo
	for _, provider := range []string{"KT", "AP"} {
		t.Run(provider, func(t *testing.T) {
			fields := repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: provider,
				SocialID: "claimed-relink-" + provider, EncryptedCredential: "old-synthetic-ciphertext"}
			if err := repo.LinkSocialIdentity(fields); err != nil {
				t.Fatal(err)
			}
			goldenExec(t, s.db, `UPDATE WEO_MEMBER_SOCIAL SET NMS_STATUS='DISCONNECTING' WHERE USR_SEQ=? AND NMS_GATE=?`, goldenMemberID, provider)
			result, err := s.db.Exec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
                (USR_SEQ,PROVIDER,ACTION,STATUS,ATTEMPT_COUNT,NEXT_ATTEMPT_AT,CLAIM_TOKEN,CREATED_AT,UPDATED_AT)
                VALUES (?,?,'DISCONNECT','PENDING',0,NOW(),'current-claim',NOW(),NOW())`, goldenMemberID, provider)
			if err != nil {
				t.Fatal(err)
			}
			id, err := result.LastInsertId()
			if err != nil {
				t.Fatal(err)
			}
			entry := model.SocialRevocationOutboxEntry{OutboxID: id, USRSeq: goldenMemberID, Provider: provider, Action: "DISCONNECT", Status: "PENDING"}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			calls := 0
			if _, err := repo.RevokeClaimedSocialDisconnect(ctx, entry, "expired-claim", func(string) error { calls++; return nil }); !errors.Is(err, repository.ErrSocialRevocationClaimExpired) {
				t.Fatalf("wrong claim must be rejected: %v", err)
			}
			if calls != 0 {
				t.Fatal("an expired worker called the provider")
			}

			entered, release, done := make(chan string, 1), make(chan struct{}), make(chan error, 1)
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			go func() {
				_, err := repo.RevokeClaimedSocialDisconnect(ctx, entry, "current-claim", func(encrypted string) error {
					entered <- encrypted
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})
				done <- err
			}()
			select {
			case credential := <-entered:
				if credential != fields.EncryptedCredential {
					t.Fatal("worker read the wrong credential")
				}
			case <-ctx.Done():
				t.Fatal("worker did not enter provider call")
			}
			fields.EncryptedCredential = "replacement-synthetic-ciphertext"
			linkResult := make(chan error, 1)
			go func() { linkResult <- repo.LinkSocialIdentity(fields) }()
			select {
			case err := <-linkResult:
				t.Fatalf("relink escaped the in-flight provider lock: %v", err)
			case <-time.After(150 * time.Millisecond):
			}
			close(release)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if err := <-linkResult; err == nil {
				t.Fatal("pending finalization must block relinking")
			}
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE OUTBOX_ID=? AND STATUS='REVOKED'`, id)
			// A replayed PENDING snapshot sees the persisted checkpoint and skips the provider.
			if _, err := repo.RevokeClaimedSocialDisconnect(ctx, entry, "current-claim", func(string) error { calls++; return nil }); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("checkpoint replay called provider again")
			}
			if err := repo.FinalizeClaimedSocialDisconnect(entry, "expired-claim"); !errors.Is(err, repository.ErrSocialRevocationClaimExpired) {
				t.Fatal(err)
			}
			if err := repo.FinalizeClaimedSocialDisconnect(entry, "current-claim"); err != nil {
				t.Fatal(err)
			}
			if err := repo.LinkSocialIdentity(fields); err != nil {
				t.Fatal(err)
			}
			if _, err := repo.RevokeClaimedSocialDisconnect(ctx, entry, "current-claim", func(string) error { calls++; return nil }); !errors.Is(err, repository.ErrSocialRevocationClaimExpired) {
				t.Fatal(err)
			}
			if err := repo.FinalizeClaimedSocialDisconnect(entry, "current-claim"); !errors.Is(err, repository.ErrSocialRevocationClaimExpired) {
				t.Fatal(err)
			}
			if err := repo.MarkSocialRevocationFailed(id, "stale synthetic failure", 1, 10, time.Now(), "PENDING", "current-claim"); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("stale worker revoked the new connection")
			}
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL WHERE USR_SEQ=? AND PROVIDER=? AND ENCRYPTED_CREDENTIAL=?`, goldenMemberID, provider, fields.EncryptedCredential)
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE OUTBOX_ID=? AND STATUS='DELIVERED'`, id)
		})
	}
}

func TestSocialRelinkBoundariesOnMariaDB(t *testing.T) {
	s := newGoldenServer(t)
	seedGoldenMember(t, s.db)
	seedGoldenMemberAs(t, s.db, goldenMemberSeed{seq: goldenMemberID + 1, usrID: "relink_other", name: "Synthetic other", phone: "01000000003", email: "other@example.test"})
	s.deps.authRepo.EnableCanonicalIdentityWrites()
	repo := s.deps.authRepo

	for _, provider := range []string{"KT", "AP"} {
		t.Run(provider+" reconnect preserves native account and replaces credential", func(t *testing.T) {
			subject := "verified-relink-" + provider
			fields := repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: provider, SocialID: subject, EncryptedCredential: "old-synthetic-ciphertext"}
			if err := repo.LinkSocialIdentity(fields); err != nil {
				t.Fatal(err)
			}
			if err := repo.DeleteSocialConnection(goldenMemberID, provider); err != nil {
				t.Fatal(err)
			}
			fields.EncryptedCredential = "new-synthetic-ciphertext"
			for i := 0; i < 3; i++ {
				if err := repo.LinkSocialIdentity(fields); err != nil {
					t.Fatal(err)
				}
				canonical := map[string]string{"KT": "KAKAO", "AP": "APPLE"}[provider]
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=? AND PROVIDER=? AND STATUS='ACTIVE' AND REVOKED_AT IS NULL`, goldenMemberID, canonical)
				goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL WHERE USR_SEQ=? AND PROVIDER=? AND ENCRYPTED_CREDENTIAL=?`, goldenMemberID, provider, fields.EncryptedCredential)
				if err := repo.DeleteSocialConnection(goldenMemberID, provider); err != nil {
					t.Fatal(err)
				}
			}
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_PASSWORD_CREDENTIAL WHERE IDENTITY_ID=? AND STATUS='ACTIVE'`, goldenMemberID)
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_PHONE_CLAIM WHERE ACCOUNT_ID=?`, goldenMemberID)
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ=? AND USR_ID='golden_member' AND USR_STATUS='CCC'`, goldenMemberID)
		})
	}

	for _, status := range []string{"ACTIVE", "REVOKED", "DISABLED"} {
		t.Run(status+" identity cannot be taken from another account", func(t *testing.T) {
			subject := "owned-social-" + status
			fields := repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: "KT", SocialID: subject}
			if err := repo.LinkSocialIdentity(fields); err != nil {
				t.Fatal(err)
			}
			if status != "ACTIVE" {
				if err := repo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
					t.Fatal(err)
				}
				goldenExec(t, s.db, `UPDATE AUTH_IDENTITY SET STATUS=? WHERE PROVIDER='KAKAO' AND SUBJECT_KEY=?`, status, subject)
			}
			fields.USRSeq = goldenMemberID + 1
			if err := repo.LinkSocialIdentity(fields); !errors.Is(err, repository.ErrSocialIdentityAlreadyLinked) {
				t.Fatalf("ownership must remain unchanged: %v", err)
			}
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=?`, fields.USRSeq)
			goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE PROVIDER='KAKAO' AND SUBJECT_KEY=? AND ACCOUNT_ID=? AND STATUS=?`, subject, goldenMemberID, status)
			if status == "ACTIVE" {
				if err := repo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
					t.Fatal(err)
				}
			}
		})
	}

	t.Run("pending disconnect rolls back connection and credential", func(t *testing.T) {
		goldenExec(t, s.db, `INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX (USR_SEQ,PROVIDER,ACTION,STATUS,ATTEMPT_COUNT,NEXT_ATTEMPT_AT,CLAIM_TOKEN,CREATED_AT,UPDATED_AT) VALUES (?,'KT','DISCONNECT','PENDING',0,NOW(),'test-claim',NOW(),NOW())`, goldenMemberID)
		fields := repository.SocialAccountFields{USRSeq: goldenMemberID, Provider: "KT", SocialID: "pending-disconnect-social", EncryptedCredential: "new-synthetic-ciphertext"}
		for _, status := range []string{"PENDING", "REVOKED", "FAILED", "FINALIZE_FAILED"} {
			goldenExec(t, s.db, `UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS=? WHERE USR_SEQ=? AND PROVIDER='KT'`, status, goldenMemberID)
			if err := repo.LinkSocialIdentity(fields); !errors.Is(err, repository.ErrSocialDisconnectPending) {
				t.Fatalf("%s must block relink: %v", status, err)
			}
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=? AND NMS_GATE='KT'`, goldenMemberID)
			goldenCount(t, s.db, 0, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL WHERE USR_SEQ=? AND PROVIDER='KT'`, goldenMemberID)
		}
		goldenExec(t, s.db, `UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX SET STATUS='DELIVERED' WHERE USR_SEQ=? AND PROVIDER='KT'`, goldenMemberID)
		if err := repo.LinkSocialIdentity(fields); err != nil {
			t.Fatal(err)
		}
		// A stale finalize must not delete the newly active connection.
		if err := repo.FinalizeSocialDisconnect(goldenMemberID, "KT"); err == nil {
			t.Fatal("stale finalize unexpectedly succeeded")
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=? AND NMS_GATE='KT' AND NMS_STATUS='ACTIVE'`, goldenMemberID)
		if err := repo.DeleteSocialConnection(goldenMemberID, "KT"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("concurrent verified links have one owner", func(t *testing.T) {
		subject := "same-subject-concurrent-link"
		social := service.NewSocialAuthService(s.deps.authService, nil, nil, nil, authQASocialVerifier{subject: subject})
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, account := range []int{goldenMemberID, goldenMemberID + 1} {
			go func(id int) {
				<-start
				_, err := social.LinkIdentity(context.Background(), id, "kakao", model.KakaoAuthorization{AccessToken: "synthetic-token"})
				results <- err
			}(account)
		}
		close(start)
		success, conflicts := 0, 0
		for i := 0; i < 2; i++ {
			err := <-results
			if err == nil {
				success++
			} else if errors.Is(err, service.ErrSocialAccountAlreadyLinked) {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflicts != 1 {
			t.Fatalf("success=%d conflicts=%d", success, conflicts)
		}
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE PROVIDER='KAKAO' AND SUBJECT_KEY=? AND STATUS='ACTIVE'`, subject)
		goldenCount(t, s.db, 1, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_GATE='KT' AND NMS_ID=?`, subject)
	})
}
