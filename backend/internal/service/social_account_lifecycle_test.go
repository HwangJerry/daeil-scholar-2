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
	expectDisconnectReservation(mock, 42, "KT", false, true, false, "KT")
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
	expectDisconnectReservation(mock, 42, "KT", false, false, false, "AP")
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

	expectDisconnectReservation(mock, 42, "KT", true, true, true, "KT")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	expectDisconnectRevokedState(mock, 42, "KT")
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

	expectDisconnectReservation(mock, 42, "KT", false, true, true, "KT", "AP")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	expectDisconnectRevokedState(mock, 42, "KT")
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

	expectDisconnectReservation(mock, 42, "KT", true, true, true, "KT")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).
		WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_SEQ = \? FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(sqlmock.AnyArg(), 42, "KT").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL[\s\S]*SET NMS_STATUS = 'ACTIVE'`).
		WithArgs(42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if _, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao); err == nil {
		t.Fatal("provider failure must be returned")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectFinalizePendingSkipsProviderAndCompletesLocalDeletion(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	lifecycle := NewSocialAccountLifecycleService(auth, nil)

	expectDisconnectReservationStatus(mock, 42, "KT", "FINALIZE_PENDING", true)
	expectConnectionDeleteTransaction(mock, 42, "KT")
	expectConnections(mock, 42, true)

	result, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectCompleted {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectRevokeSuccessThenFinalizeFailureRetriesOnlyLocalDeletion(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	configureCredentialVaultForTest(auth)
	lifecycle := NewSocialAccountLifecycleService(auth, nil)
	encryptedCredential, err := lifecycle.EncryptCredential("provider-credential")
	if err != nil {
		t.Fatal(err)
	}
	auth.httpClient = successfulKakaoUnlinkClient()

	expectDisconnectReservation(mock, 42, "KT", true, true, true, "KT")
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	expectDisconnectRevokedState(mock, 42, "KT")
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).WithArgs(42, "KT").
		WillReturnError(errors.New("local finalize unavailable"))
	mock.ExpectRollback()

	if _, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao); err == nil {
		t.Fatal("local finalize failure must be returned")
	}

	expectDisconnectReservationStatus(mock, 42, "KT", "FINALIZE_PENDING", true)
	expectConnectionDeleteTransaction(mock, 42, "KT")
	expectConnections(mock, 42, true)

	result, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != model.SocialDisconnectCompleted {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDisconnectRetryFailureKeepsConnectionNonActive(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	configureCredentialVaultForTest(auth)
	lifecycle := NewSocialAccountLifecycleService(auth, nil)
	encryptedCredential, err := lifecycle.EncryptCredential("provider-credential")
	if err != nil {
		t.Fatal(err)
	}
	auth.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("already revoked or unavailable")
	})}

	expectDisconnectReservationStatus(mock, 42, "KT", "DISCONNECTING", true)
	mock.ExpectQuery(`SELECT ENCRYPTED_CREDENTIAL`).WithArgs(42, "KT").
		WillReturnRows(sqlmock.NewRows([]string{"ENCRYPTED_CREDENTIAL"}).AddRow(encryptedCredential))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_SEQ = \? FOR UPDATE`).WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(42))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).WithArgs(sqlmock.AnyArg(), 42, "KT").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if _, err := lifecycle.Disconnect(context.Background(), 42, model.SocialProviderKakao); err == nil {
		t.Fatal("provider failure must be returned")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectConnections(mock sqlmock.Sqlmock, usrSeq int, hasPassword bool, providers ...string) {
	mock.ExpectQuery(`FROM AUTH_IDENTITY i[\s\S]*AUTH_PASSWORD_CREDENTIAL c[\s\S]*i.STATUS = 'ACTIVE'[\s\S]*c.STATUS = 'ACTIVE'`).
		WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"HAS_PASSWORD"}).AddRow(hasPassword))
	rows := sqlmock.NewRows([]string{"NMS_GATE"})
	for _, provider := range providers {
		rows.AddRow(provider)
	}
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS = 'ACTIVE'`).
		WithArgs(usrSeq).
		WillReturnRows(rows)
}

func expectDisconnectReservation(mock sqlmock.Sqlmock, usrSeq int, provider string, hasPassword, connected, allowed bool, providers ...string) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).
		WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(usrSeq))
	statusRows := sqlmock.NewRows([]string{"NMS_STATUS"})
	if connected {
		statusRows.AddRow("ACTIVE")
	}
	mock.ExpectQuery(`SELECT NMS_STATUS`).WithArgs(usrSeq, provider).WillReturnRows(statusRows)
	expectConnections(mock, usrSeq, hasPassword, providers...)
	if !connected {
		mock.ExpectCommit()
		return
	}
	if !allowed {
		mock.ExpectRollback()
		return
	}
	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL[\s\S]*SET NMS_STATUS = 'DISCONNECTING'`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(usrSeq, provider, usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

func expectDisconnectReservationStatus(mock sqlmock.Sqlmock, usrSeq int, provider, status string, hasPassword bool) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ[\s\S]*FROM WEO_MEMBER[\s\S]*FOR UPDATE`).WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(usrSeq))
	mock.ExpectQuery(`SELECT NMS_STATUS`).WithArgs(usrSeq, provider).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_STATUS"}).AddRow(status))
	expectConnections(mock, usrSeq, hasPassword)
	mock.ExpectCommit()
}

func expectDisconnectRevokedState(mock sqlmock.Sqlmock, usrSeq int, provider string) {
	mock.ExpectExec(`UPDATE WEO_MEMBER_SOCIAL[\s\S]*SET NMS_STATUS = 'FINALIZE_PENDING'`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectConnectionDeleteTransaction(mock sqlmock.Sqlmock, usrSeq int, provider string) {
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(usrSeq, provider).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
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
