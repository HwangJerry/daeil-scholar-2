package service

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
)

func TestDisconnectRejectsLastLoginMethod(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	expectConnections(mock, 42, false, "KT")
	lifecycle := NewSocialAccountLifecycleService(auth, nil)

	_, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)

	if !errors.Is(err, ErrLastLoginMethod) {
		t.Fatalf("disconnect error = %v, want LAST_LOGIN_METHOD", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectUnlinkedProviderIsIdempotentWithoutOutbox(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	expectConnections(mock, 42, false, "AP")
	lifecycle := NewSocialAccountLifecycleService(auth, nil)

	result, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectNotConnected || len(result.Connections.Providers) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectAllowsPasswordAlternativeAndDeletesAtomically(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	configureCredentialVaultForTest(auth)
	lifecycle := NewSocialAccountLifecycleService(auth, nil)
	encryptedCredential, err := lifecycle.EncryptCredential("provider-credential")
	if err != nil {
		t.Fatal(err)
	}
	auth.httpClient = successfulKakaoUnlinkClient()

	expectConnections(mock, 42, true, "KT")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	expectConnectionDeleteTransaction(mock, 42, "KT")
	expectConnections(mock, 42, true)

	result, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)

	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectCompleted ||
		len(result.Connections.Providers) != 0 ||
		!result.Connections.HasPassword {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectAllowsAnotherProvider(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	configureCredentialVaultForTest(auth)
	lifecycle := NewSocialAccountLifecycleService(auth, nil)
	encryptedCredential, err := lifecycle.EncryptCredential("provider-credential")
	if err != nil {
		t.Fatal(err)
	}
	auth.httpClient = successfulKakaoUnlinkClient()

	expectConnections(mock, 42, false, "KT", "AP")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	expectConnectionDeleteTransaction(mock, 42, "KT")
	expectConnections(mock, 42, false, "AP")

	result, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)

	if err != nil {
		t.Fatal(err)
	}
	if len(result.Connections.Providers) != 1 ||
		result.Connections.Providers[0] != model.SocialProviderApple {
		t.Fatalf("connections = %#v", result.Connections)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectRevokeFailurePreservesConnectionAndRecordsOutbox(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	configureCredentialVaultForTest(auth)
	lifecycle := NewSocialAccountLifecycleService(auth, nil)
	encryptedCredential, err := lifecycle.EncryptCredential("provider-credential")
	if err != nil {
		t.Fatal(err)
	}
	auth.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("provider unavailable")
	})}

	expectConnections(mock, 42, true, "KT")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "KT", revocationActionDisconnect, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao); err == nil {
		t.Fatal("provider failure must be returned")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectConnections(mock sqlmock.Sqlmock, usrSeq int, hasPassword bool, providers ...string) {
	mock.ExpectQuery(`SELECT CASE WHEN IFNULL\(USR_PWD, ''\)`).
		WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"HAS_PASSWORD"}).AddRow(hasPassword))
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(usrSeq).
		WillReturnRows(rows)
}

func expectConnectionDeleteTransaction(mock sqlmock.Sqlmock, usrSeq int, provider string) {
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}

func configureCredentialVaultForTest(auth *AuthService) {
	auth.cfg.Apple.CredentialEncryptionKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func successfulKakaoUnlinkClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
		}, nil
	})}
}
