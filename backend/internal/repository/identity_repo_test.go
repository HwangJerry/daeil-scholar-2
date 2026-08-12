package repository

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestIdentityRepositoryUpsertIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewIdentityRepository(sqlx.NewDb(db, "sqlmock"))
	normalizedEmail := "member@example.com"
	identity := model.Identity{
		AccountSeq:      42,
		Provider:        model.IdentityProviderEmail,
		SubjectKey:      "member@example.com",
		NormalizedEmail: &normalizedEmail,
	}

	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
		WithArgs(identity.AccountSeq, string(identity.Provider), identity.SubjectKey, &normalizedEmail).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpsertIdentity(identity); err != nil {
		t.Fatalf("UpsertIdentity() = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityRepositoryFindIdentityByProviderSubject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewIdentityRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL`).
		WithArgs(model.IdentityProviderKakao, "subject-1").
		WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_ID", "ACCOUNT_ID", "PROVIDER", "SUBJECT_KEY", "NORMALIZED_EMAIL"}).
			AddRow(int64(7), 123, "KAKAO", "subject-1", nil))

	identity, err := repo.FindIdentityByProviderSubject(model.IdentityProviderKakao, "subject-1")
	if err != nil {
		t.Fatalf("FindIdentityByProviderSubject() = %v", err)
	}
	if identity == nil {
		t.Fatal("identity is nil")
	}
	if identity.IdentityID != 7 || identity.AccountSeq != 123 || identity.Provider != model.IdentityProviderKakao {
		t.Fatalf("identity = %#v", identity)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityRepositoryFindIdentityMissingReturnsNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewIdentityRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL`).
		WithArgs(99).
		WillReturnError(sql.ErrNoRows)

	identity, err := repo.FindIdentity(99)
	if err != nil {
		t.Fatalf("FindIdentity() = %v", err)
	}
	if identity != nil {
		t.Fatalf("identity = %#v", identity)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityRepositoryListIdentities(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewIdentityRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_ID", "ACCOUNT_ID", "PROVIDER", "SUBJECT_KEY", "NORMALIZED_EMAIL"}).
			AddRow(int64(1), 42, "EMAIL", "email@example.com", "email@example.com").
			AddRow(int64(2), 42, "KAKAO", "kakao-subject", nil))

	identities, err := repo.ListIdentities(42)
	if err != nil {
		t.Fatalf("ListIdentities() = %v", err)
	}
	if len(identities) != 2 {
		t.Fatalf("len(ListIdentities()) = %d, want 2", len(identities))
	}
	if identities[0].IdentityID != 1 || identities[1].IdentityID != 2 {
		t.Fatalf("identities = %#v", identities)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIdentityRepositoryDisableAndDeleteIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewIdentityRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`UPDATE AUTH_IDENTITY SET STATUS = 'DISABLED'`).
		WithArgs(555).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM AUTH_IDENTITY`).
		WithArgs(555).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.DisableIdentity(555); err != nil {
		t.Fatalf("DisableIdentity() = %v", err)
	}
	if err := repo.DeleteIdentity(555); err != nil {
		t.Fatalf("DeleteIdentity() = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
