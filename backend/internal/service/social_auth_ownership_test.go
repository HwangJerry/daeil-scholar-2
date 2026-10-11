package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

func TestSocialAuthRejectsReservedOwnershipBeforeContinuation(t *testing.T) {
	for _, provider := range []model.SocialProvider{model.SocialProviderKakao, model.SocialProviderApple} {
		for _, tc := range []struct{ identity, member, code string }{
			{"REVOKED", "CCC", "SOCIAL_ACCOUNT_RELINK_REQUIRED"},
			{"REVOKED", "BBB", "SOCIAL_ACCOUNT_RELINK_REQUIRED"},
			{"REVOKED", "AAA", "ACCOUNT_WITHDRAWN"},
			{"REVOKED", "DDD", "ACCOUNT_SUSPENDED"},
			{"DISABLED", "CCC", "ACCOUNT_SUSPENDED"},
			{"REVOKED", "", "ACCOUNT_SUSPENDED"},
			{"UNKNOWN", "CCC", "ACCOUNT_SUSPENDED"},
		} {
			t.Run(string(provider)+"/"+tc.identity+"/"+tc.member, func(t *testing.T) {
				auth, mock, cleanup := newAuthServiceForTest(t)
				defer cleanup()
				auth.repo.EnableCanonicalIdentityWrites()
				canonical := map[model.SocialProvider]string{model.SocialProviderKakao: "KAKAO", model.SocialProviderApple: "APPLE"}[provider]
				mock.ExpectQuery(`SELECT i.STATUS AS IDENTITY_STATUS`).WithArgs(canonical, "synthetic-subject").
					WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_STATUS", "USR_STATUS"}).AddRow(tc.identity, tc.member))
				store := cache.New(time.Minute, time.Minute)
				social := NewSocialAuthService(auth, nil, NewSocialLinkTokenStore(store), nil,
					stubSocialVerifier{account: VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: provider, Subject: "synthetic-subject"}, ProviderToken: "synthetic-provider-secret"}})
				result, err := social.Authenticate(context.Background(), socialAuthorizationForTest(provider))
				if err != nil {
					t.Fatal(err)
				}
				if result.Status != model.SocialAuthRejected || result.Rejected == nil || result.Rejected.Code != tc.code {
					t.Fatalf("status=%s rejection=%v, want %s", result.Status, result.Rejected, tc.code)
				}
				if tc.code == "SOCIAL_ACCOUNT_RELINK_REQUIRED" && result.Rejected.Message != "기존 로그인 방법으로 로그인한 뒤 계정 설정에서 같은 소셜 계정을 다시 연결해주세요." {
					t.Fatal("recovery message must support accounts without ID/password login")
				}
				if result.Session != nil || result.LinkRequired != nil || store.ItemCount() != 0 {
					t.Fatal("rejected ownership created a session or continuation")
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestSocialAuthProviderFailurePrecedesOwnershipLookup(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	auth.repo.EnableCanonicalIdentityWrites()
	proofErr := errors.New("synthetic provider verification failure")
	store := cache.New(time.Minute, time.Minute)
	social := NewSocialAuthService(auth, nil, NewSocialLinkTokenStore(store), nil, stubSocialVerifier{account: VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: model.SocialProviderKakao}}, err: proofErr})
	if result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{}); !errors.Is(err, proofErr) || result.Session != nil || result.LinkRequired != nil || store.ItemCount() != 0 {
		t.Fatal("failed provider proof read ownership or created continuation")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialAuthOwnershipLookupFailureDoesNotStartSignup(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	auth.repo.EnableCanonicalIdentityWrites()
	lookupErr := errors.New("synthetic ownership lookup failure")
	mock.ExpectQuery(`SELECT i.STATUS AS IDENTITY_STATUS`).WithArgs("KAKAO", "synthetic-subject").WillReturnError(lookupErr)
	store := cache.New(time.Minute, time.Minute)
	social := NewSocialAuthService(auth, nil, NewSocialLinkTokenStore(store), nil, stubSocialVerifier{account: VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: model.SocialProviderKakao, Subject: "synthetic-subject"}}})
	result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{})
	if !errors.Is(err, lookupErr) || result.Session != nil || result.LinkRequired != nil || store.ItemCount() != 0 {
		t.Fatal("ownership lookup failure did not fail closed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialAuthUnownedCanonicalIdentityStillStartsSignup(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	auth.repo.EnableCanonicalIdentityWrites()
	mock.ExpectQuery(`SELECT i.STATUS AS IDENTITY_STATUS`).WithArgs("KAKAO", "synthetic-new-subject").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).WithArgs("KT", "synthetic-new-subject").WillReturnError(sql.ErrNoRows)
	social := NewSocialAuthService(auth, nil, NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute)), nil, stubSocialVerifier{account: VerifiedSocialAccount{Identity: model.VerifiedSocialIdentity{Provider: model.SocialProviderKakao, Subject: "synthetic-new-subject"}}})
	result, err := social.Authenticate(context.Background(), model.KakaoAuthorization{})
	if err != nil || result.Status != model.SocialAuthLinkRequired || result.LinkRequired == nil {
		t.Fatalf("new identity did not start signup: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
