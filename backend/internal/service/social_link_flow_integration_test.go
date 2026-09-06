package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// Set SOCIAL_LINK_TEST_DSN to a disposable MariaDB database. All fixtures use
// connection-local temporary tables, including the production identity indexes.
func TestSocialSignUpConflictThenLinkAndLoginWithSameEmail(t *testing.T) {
	for _, existingProvider := range []string{"EMAIL", "LOCAL_USERNAME", "KAKAO"} {
		for _, provider := range []model.SocialProvider{model.SocialProviderApple, model.SocialProviderKakao} {
			t.Run(existingProvider+"/"+string(provider), func(t *testing.T) {
				db := socialLinkIntegrationDB(t)
				_, err := db.Exec(`INSERT INTO AUTH_IDENTITY
					(ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS)
					VALUES (42, ?, 'existing-login', 'member@example.com', 'ACTIVE')`, existingProvider)
				if err != nil {
					t.Fatal(err)
				}
				repo := repository.NewAuthRepository(db)
				repo.EnableCanonicalIdentityWrites()
				cfg := &config.Config{JWT: config.JWTConfig{
					Secret: "integration-test-only", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
				}}
				cacheStore := cache.New(time.Minute, time.Minute)
				auth := NewAuthService(repo, repository.NewSessionRepository(db), cfg, cacheStore, zerolog.Nop())
				tokens := NewSocialLinkTokenStore(cacheStore)
				verifier := stubSocialVerifier{account: VerifiedSocialAccount{
					Identity: model.VerifiedSocialIdentity{
						Provider: provider, Subject: "new-social-subject", Email: "member@example.com", EmailVerified: true,
					},
				}}
				social := NewSocialAuthService(auth, NewMobileSessionIssuer(auth), tokens, nil, verifier)
				var authorization model.SocialAuthorization = model.AppleAuthorization{
					ChallengeID: "verified-challenge", IdentityToken: "verified-token", AuthorizationCode: "verified-code",
				}
				pathProvider := "apple"
				if provider == model.SocialProviderKakao {
					authorization = model.KakaoAuthorization{AccessToken: "verified-token"}
					pathProvider = "kakao"
				}
				ctx := context.Background()
				result, err := social.Authenticate(ctx, authorization)
				if err != nil || result.Status != model.SocialAuthLinkRequired {
					t.Fatalf("first login = %v, %v", result.Status, err)
				}
				_, _, err = auth.LinkSocialAccount(SocialLinkParams{
					Provider: string(provider), SocialID: "new-social-subject", Name: "Member",
					Phone: "01012345678", Email: "member@example.com",
				}, NewMemberService(repo))
				if !errors.Is(err, ErrOwnershipConfirmationRequired) {
					t.Fatalf("duplicate phone must require existing-account authentication: %v", err)
				}
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM WEO_MEMBER`, 1)
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL`, 0)
				// The authenticated endpoint supplies account 42 only after checking
				// the existing account's session. Apple verification is stubbed here.
				connections, err := social.LinkIdentity(ctx, 42, pathProvider, authorization)
				if err != nil || len(connections.Providers) != 1 || !connections.HasPassword {
					t.Fatalf("link to existing account = %+v, %v", connections, err)
				}
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE ACCOUNT_ID=42`, 2)
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM AUTH_IDENTITY WHERE SUBJECT_KEY='new-social-subject' AND NORMALIZED_EMAIL IS NULL`, 1)
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE NMS_EMAIL='member@example.com'`, 1)
				if _, err := social.LinkIdentity(ctx, 42, pathProvider, authorization); err != nil {
					t.Fatalf("repeat link must be idempotent: %v", err)
				}
				if _, err := social.LinkIdentity(ctx, 99, pathProvider, authorization); !errors.Is(err, ErrSocialAccountAlreadyLinked) {
					t.Fatalf("another account must not take the identity: %v", err)
				}
				result, err = social.Authenticate(ctx, authorization)
				if err != nil || result.Status != model.SocialAuthAuthenticated || result.Session.User.USRSeq != 42 {
					t.Fatalf("subsequent social login must use the existing account: %+v, %v", result, err)
				}
				assertSocialLinkCount(t, db, `SELECT COUNT(*) FROM WEO_MEMBER`, 1)
			})
		}
	}
}

func socialLinkIntegrationDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("SOCIAL_LINK_TEST_DSN")
	if dsn == "" {
		t.Skip("set SOCIAL_LINK_TEST_DSN to run the MariaDB social linking regression")
	}
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TEMPORARY TABLE WEO_MEMBER (
			USR_SEQ INT PRIMARY KEY, USR_ID VARCHAR(100), USR_NAME VARCHAR(100),
			USR_STATUS CHAR(3), USR_PHONE VARCHAR(20), USR_FN VARCHAR(10),
			USR_EMAIL VARCHAR(255), USR_NICK VARCHAR(100), USR_PHOTO VARCHAR(255), USR_PWD VARCHAR(255)
		) ENGINE=InnoDB`,
		`CREATE TEMPORARY TABLE WEO_MEMBER_SOCIAL (
			SEQ INT AUTO_INCREMENT PRIMARY KEY, USR_SEQ INT NOT NULL,
			NMS_GATE CHAR(2) NOT NULL, NMS_ID VARCHAR(50) NOT NULL,
			NMS_EMAIL VARCHAR(255), NMS_STATUS VARCHAR(20) DEFAULT 'ACTIVE', REG_DATE DATETIME,
			UNIQUE KEY UK_USR_PROVIDER (USR_SEQ,NMS_GATE),
			UNIQUE KEY UK_PROVIDER_SUBJECT (NMS_GATE,NMS_ID)
		) ENGINE=InnoDB`,
		`CREATE TEMPORARY TABLE AUTH_IDENTITY (
			IDENTITY_ID BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			ACCOUNT_ID INT NOT NULL, PROVIDER VARCHAR(20) NOT NULL,
			SUBJECT_KEY VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
			NORMALIZED_EMAIL VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NULL,
			STATUS VARCHAR(20), VERIFIED_AT DATETIME, CREATED_AT DATETIME, UPDATED_AT DATETIME,
			UNIQUE KEY UQ_AUTH_IDENTITY_PROVIDER_SUBJECT (PROVIDER,SUBJECT_KEY),
			UNIQUE KEY UQ_AUTH_IDENTITY_NORMALIZED_EMAIL (NORMALIZED_EMAIL)
		) ENGINE=InnoDB`,
		`CREATE TEMPORARY TABLE ALUMNI_VERIFICATION (
			USR_SEQ INT, STATUS VARCHAR(20), GRADUATION_YEAR INT, COHORT VARCHAR(10),
			DEPARTMENT VARCHAR(20), REJECTION_REASON TEXT, SUBMITTED_AT DATETIME, REVIEWED_AT DATETIME
		) ENGINE=InnoDB`,
		`CREATE TEMPORARY TABLE ALUMNI_ADMIN_ROLE (USR_SEQ INT, ADMIN_ROLE VARCHAR(20)) ENGINE=InnoDB`,
		`CREATE TEMPORARY TABLE ALUMNI_MOBILE_REFRESH_TOKEN (
			MRT_JTI VARCHAR(100), USR_SEQ INT, MRT_SID VARCHAR(100), EXPIRES_AT DATETIME, CREATED_AT DATETIME
		) ENGINE=InnoDB`,
		`INSERT INTO WEO_MEMBER (USR_SEQ,USR_ID,USR_NAME,USR_STATUS,USR_PHONE,USR_EMAIL,USR_PWD)
			VALUES (42,'member','Member','CCC','01012345678','member@example.com','existing-password')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func assertSocialLinkCount(t *testing.T, db *sqlx.DB, query string, want int) {
	t.Helper()
	var count int
	if err := db.Get(&count, query); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%s = %d, want %d", query, count, want)
	}
}
