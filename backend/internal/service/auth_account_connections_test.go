package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestLinkIdentityConnectsKakaoAndApple(t *testing.T) {
	tests := []struct {
		name          string
		pathProvider  string
		provider      model.SocialProvider
		authorization model.SocialAuthorization
	}{
		{name: "kakao", pathProvider: "kakao", provider: model.SocialProviderKakao, authorization: model.KakaoAuthorization{AccessToken: "kakao-token"}},
		{name: "apple", pathProvider: "apple", provider: model.SocialProviderApple, authorization: model.AppleAuthorization{ChallengeID: "challenge", IdentityToken: "identity", AuthorizationCode: "code"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth, mock, cleanup := newAuthServiceForTest(t)
			defer cleanup()
			mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
				WithArgs(string(test.provider), "provider-subject").
				WillReturnError(sql.ErrNoRows)
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
				WithArgs(42, string(test.provider), "provider-subject", "member@example.com").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			expectAccountConnections(mock, []string{string(test.provider)}, true)

			social := NewSocialAuthService(
				auth,
				nil,
				nil,
				nil,
				stubSocialVerifier{account: VerifiedSocialAccount{
					Identity: model.VerifiedSocialIdentity{
						Provider: test.provider,
						Subject:  "provider-subject",
						Email:    "member@example.com",
					},
				}},
			)
			connections, err := social.LinkIdentity(context.Background(), 42, test.pathProvider, test.authorization)
			if err != nil {
				t.Fatal(err)
			}
			if len(connections.Providers) != 1 || connections.Providers[0] != string(test.provider) || !connections.HasPassword {
				t.Fatalf("connections = %#v", connections)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLinkIdentityIsIdempotentForSameMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnRows(memberRowsForSequence(42))
	expectAccountConnections(mock, []string{"KT"}, false)
	social := newLinkIdentityService(auth, 42)

	connections, err := social.LinkIdentity(
		context.Background(),
		42,
		"kakao",
		model.KakaoAuthorization{AccessToken: "token"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections.Providers) != 1 || connections.Providers[0] != "KT" {
		t.Fatalf("connections = %#v", connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityRejectsIdentityLinkedToAnotherMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnRows(memberRowsForSequence(99))
	social := newLinkIdentityService(auth, 42)

	_, err := social.LinkIdentity(
		context.Background(),
		42,
		"kakao",
		model.KakaoAuthorization{AccessToken: "token"},
	)
	if !errors.Is(err, ErrSocialAccountAlreadyLinked) {
		t.Fatalf("error = %v, want ErrSocialAccountAlreadyLinked", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityReturnsSuccessWhenConcurrentLinkWinnerIsSameMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "provider-subject", "").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "concurrent link won"})
	mock.ExpectRollback()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnRows(memberRowsForSequence(42))
	expectAccountConnections(mock, []string{"KT"}, false)
	social := newLinkIdentityService(auth, 42)

	connections, err := social.LinkIdentity(
		context.Background(),
		42,
		"kakao",
		model.KakaoAuthorization{AccessToken: "token"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections.Providers) != 1 || connections.Providers[0] != "KT" {
		t.Fatalf("connections = %#v", connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityRejectsWhenConcurrentLinkWinnerIsAnotherMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "provider-subject", "").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "concurrent link won"})
	mock.ExpectRollback()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnRows(memberRowsForSequence(99))
	social := newLinkIdentityService(auth, 42)

	_, err := social.LinkIdentity(
		context.Background(),
		42,
		"kakao",
		model.KakaoAuthorization{AccessToken: "token"},
	)
	if !errors.Is(err, ErrSocialAccountAlreadyLinked) {
		t.Fatalf("error = %v, want ErrSocialAccountAlreadyLinked", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityPropagatesCanonicalIdentityInsertFailureAfterRollback(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	auth.repo.EnableCanonicalIdentityWrites()
	canonicalInsertErr := errors.New("canonical identity insert failed")

	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "provider-subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT", "provider-subject", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
		WithArgs(42, string(model.IdentityProviderKakao), "provider-subject", nil).
		WillReturnError(canonicalInsertErr)
	mock.ExpectRollback()
	social := newLinkIdentityService(auth, 42)

	_, err := social.LinkIdentity(
		context.Background(),
		42,
		"kakao",
		model.KakaoAuthorization{AccessToken: "token"},
	)
	if !errors.Is(err, canonicalInsertErr) {
		t.Fatalf("error = %v, want canonical insert error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkIdentityRejectsInvalidProvider(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	social := newLinkIdentityService(auth, 42)

	_, err := social.LinkIdentity(context.Background(), 42, "naver", model.KakaoAuthorization{AccessToken: "token"})
	if !errors.Is(err, ErrInvalidSocialProvider) {
		t.Fatalf("error = %v, want ErrInvalidSocialProvider", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newLinkIdentityService(auth *AuthService, usrSeq int) *SocialAuthService {
	return NewSocialAuthService(
		auth,
		nil,
		nil,
		nil,
		stubSocialVerifier{account: VerifiedSocialAccount{
			Identity: model.VerifiedSocialIdentity{
				Provider: model.SocialProviderKakao,
				Subject:  "provider-subject",
			},
		}},
	)
}

func memberRowsForSequence(usrSeq int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
	}).AddRow(usrSeq, "member", "Member", "CCC", "01012345678", "", "member@example.com", "", nil)
}

func TestDisconnectSocialConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth := &AuthService{repo: repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))}

	expectAccountConnections(mock, []string{"AP", "KT"}, false)
	mock.ExpectBegin()
	expectLockedLoginMethods(mock, []string{"AP", "KT"}, false)
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	expectAccountConnections(mock, []string{"AP"}, false)

	result, err := auth.Disconnect(42, "kakao")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectStatusDisconnected {
		t.Fatalf("status = %q", result.Status)
	}
	if len(result.Connections.Providers) != 1 || result.Connections.Providers[0] != "AP" {
		t.Fatalf("connections = %#v", result.Connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialConnectionReturnsNotConnected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth := &AuthService{repo: repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))}
	expectAccountConnections(mock, []string{"AP"}, true)

	result, err := auth.Disconnect(42, "kakao")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectStatusNotConnected {
		t.Fatalf("status = %q", result.Status)
	}
	if len(result.Connections.Providers) != 1 || result.Connections.Providers[0] != "AP" {
		t.Fatalf("connections = %#v", result.Connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectSocialConnectionRejectsLastLoginMethod(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auth := &AuthService{repo: repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))}
	expectAccountConnections(mock, []string{"KT"}, false)
	mock.ExpectBegin()
	expectLockedLoginMethods(mock, []string{"KT"}, false)
	mock.ExpectRollback()

	_, err = auth.Disconnect(42, "kakao")
	if !errors.Is(err, ErrLastLoginMethod) {
		t.Fatalf("error = %v, want ErrLastLoginMethod", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectLockedLoginMethods(mock sqlmock.Sqlmock, providers []string, hasPassword bool) {
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(rows)
	passwordPresent := 0
	if hasPassword {
		passwordPresent = 1
	}
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(passwordPresent))
}

func expectAccountConnections(mock sqlmock.Sqlmock, providers []string, hasPassword bool) {
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE`).WithArgs(42).WillReturnRows(rows)
	passwordPresent := 0
	if hasPassword {
		passwordPresent = 1
	}
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(passwordPresent))
}
