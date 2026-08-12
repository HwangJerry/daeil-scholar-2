package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestCredentialRepositoryUpsertPasswordCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	params := "m=65537,t=2,p=1"
	cred := model.PasswordCredential{
		IdentityID:     11,
		Provider:       model.IdentityProviderEmail,
		Algorithm:      "argon2id",
		ParametersText: &params,
		PasswordHash:   "$argon2id$v=19$m=65537,t=2,p=1$...",
		Status:         model.PasswordCredentialStatusActive,
	}

	mock.ExpectExec(`INSERT INTO AUTH_PASSWORD_CREDENTIAL`).
		WithArgs(cred.IdentityID, string(cred.Provider), cred.Algorithm, cred.ParametersText, cred.PasswordHash).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpsertPasswordCredential(cred); err != nil {
		t.Fatalf("UpsertPasswordCredential() = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryFindPasswordCredentialActive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS`).
		WithArgs(11).
		WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_ID", "PROVIDER", "ALGORITHM", "PARAMETERS_TEXT", "PASSWORD_HASH", "STATUS"}).
			AddRow(int64(11), "EMAIL", "argon2id", "m=65537,t=2,p=1", "hash", "ACTIVE"))

	cred, err := repo.FindPasswordCredential(11)
	if err != nil {
		t.Fatalf("FindPasswordCredential() = %v", err)
	}
	if cred == nil {
		t.Fatal("credential is nil")
	}
	if cred.IdentityID != 11 || cred.Provider != model.IdentityProviderEmail || cred.Algorithm != "argon2id" {
		t.Fatalf("credential = %#v", cred)
	}
	if cred.ParametersText == nil || *cred.ParametersText == "" {
		t.Fatalf("credential.ParametersText = %#v", cred)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryFindPasswordCredentialMissingReturnsNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS`).
		WithArgs(100).
		WillReturnError(sql.ErrNoRows)

	cred, err := repo.FindPasswordCredential(100)
	if err != nil {
		t.Fatalf("FindPasswordCredential() = %v", err)
	}
	if cred != nil {
		t.Fatalf("cred = %#v", cred)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryFindPasswordCredentialReturnsDisabledStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS`).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_ID", "PROVIDER", "ALGORITHM", "PARAMETERS_TEXT", "PASSWORD_HASH", "STATUS"}).
			AddRow(int64(11), "LOCAL_USERNAME", "ARGON2ID", "m=19456,t=2,p=1", "disabled-hash", "DISABLED"))

	credential, err := repo.FindPasswordCredential(11)
	if err != nil {
		t.Fatalf("FindPasswordCredential() error = %v", err)
	}
	if credential == nil || credential.Status != model.PasswordCredentialStatusDisabled {
		t.Fatalf("credential = %+v, want disabled row", credential)
	}
}

func TestCredentialRepositoryRehashPasswordCredentialUsesCompareAndSwap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))
	parameters := "m=19456,t=2,p=1"
	credential := model.PasswordCredential{
		IdentityID: 11, Provider: model.IdentityProviderLocalUsername, Algorithm: model.PasswordAlgorithmArgon2id,
		ParametersText: &parameters, PasswordHash: "new-hash", Status: model.PasswordCredentialStatusActive,
	}

	mock.ExpectExec(`UPDATE AUTH_PASSWORD_CREDENTIAL`).
		WithArgs(credential.Algorithm, credential.ParametersText, credential.PasswordHash, int64(11), "old-hash").
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := repo.RehashPasswordCredential(11, "old-hash", credential)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("RehashPasswordCredential() updated = false")
	}
}

func TestCredentialRepositoryDeletePasswordCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`DELETE FROM AUTH_PASSWORD_CREDENTIAL`).
		WithArgs(11).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.DeletePasswordCredential(11); err != nil {
		t.Fatalf("DeletePasswordCredential() = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryUpsertProviderCredentialForIdentityOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))
	identityID := int64(99)
	nonce := make([]byte, 12)
	nonce[0] = 1
	cred := model.ProviderCredential{
		IdentityID: &identityID,
		Provider:   model.IdentityProviderKakao,
		KeyID:      "k1",
		NonceBytes: nonce,
		Algorithm:  "AES-256-GCM",
		Ciphertext: []byte("cipher-bytes"),
	}

	mock.ExpectExec(`INSERT INTO AUTH_PROVIDER_CREDENTIAL`).
		WithArgs(*cred.IdentityID, nil, string(cred.Provider), cred.KeyID, cred.NonceBytes, cred.Algorithm, cred.Ciphertext).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.UpsertProviderCredential(cred); err != nil {
		t.Fatalf("UpsertProviderCredential() = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryFindProviderCredentialByIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT CREDENTIAL_ID, IDENTITY_ID, CONTINUATION_TOKEN_HASH`).
		WithArgs(99, string(model.IdentityProviderApple)).
		WillReturnRows(sqlmock.NewRows([]string{"CREDENTIAL_ID", "IDENTITY_ID", "CONTINUATION_TOKEN_HASH", "PROVIDER", "KEY_ID", "NONCE_BYTES", "ALGORITHM", "CIPHERTEXT"}).
			AddRow(int64(22), int64(99), nil, "APPLE", "k2", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, "AES-256-GCM", []byte("ct")))

	cred, err := repo.FindProviderCredentialByIdentity(99, model.IdentityProviderApple)
	if err != nil {
		t.Fatalf("FindProviderCredentialByIdentity() = %v", err)
	}
	if cred == nil {
		t.Fatal("credential is nil")
	}
	if cred.Provider != model.IdentityProviderApple || cred.IdentityID == nil || *cred.IdentityID != 99 {
		t.Fatalf("cred = %#v", cred)
	}
	if len(cred.NonceBytes) != 12 {
		t.Fatalf("unexpected nonce length %d", len(cred.NonceBytes))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRepositoryFindProviderCredentialByContinuationToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewCredentialRepository(sqlx.NewDb(db, "sqlmock"))
	continuation := "ctokenhash"

	mock.ExpectQuery(`SELECT CREDENTIAL_ID, IDENTITY_ID, CONTINUATION_TOKEN_HASH`).
		WithArgs(continuation).
		WillReturnRows(sqlmock.NewRows([]string{"CREDENTIAL_ID", "IDENTITY_ID", "CONTINUATION_TOKEN_HASH", "PROVIDER", "KEY_ID", "NONCE_BYTES", "ALGORITHM", "CIPHERTEXT"}).
			AddRow(int64(23), nil, continuation, "KAKAO", "k3", make([]byte, 12), "AES-256-GCM", []byte("ct2")))

	cred, err := repo.FindProviderCredentialByContinuationToken(continuation)
	if err != nil {
		t.Fatalf("FindProviderCredentialByContinuationToken() = %v", err)
	}
	if cred == nil {
		t.Fatal("credential is nil")
	}
	if cred.ContinuationTokenHash == nil || *cred.ContinuationTokenHash != continuation {
		t.Fatalf("cred = %#v", cred)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
