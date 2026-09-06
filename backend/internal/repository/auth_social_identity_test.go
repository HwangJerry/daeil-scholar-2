package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestCreateSocialAccountWritesCanonicalIdentityWhenReady(t *testing.T) {
	tests := []struct {
		name            string
		legacyProvider  string
		canonical       model.IdentityProvider
		email           string
		normalizedEmail any
	}{
		{
			name:            "kakao keeps provider email out of email authority",
			legacyProvider:  string(model.SocialProviderKakao),
			canonical:       model.IdentityProviderKakao,
			email:           "  Member@Example.COM ",
			normalizedEmail: nil,
		},
		{
			name:            "apple keeps provider email out of email authority",
			legacyProvider:  string(model.SocialProviderApple),
			canonical:       model.IdentityProviderApple,
			email:           "Member@Example.COM",
			normalizedEmail: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			repo.EnableCanonicalIdentityWrites()
			fields := socialAccountFixture(test.legacyProvider, test.email)

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
				WillReturnResult(sqlmock.NewResult(42, 1))
			mock.ExpectExec(`INSERT INTO AUTH_ACCOUNT_STATE`).
				WithArgs(42).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
				WithArgs(42, model.VerificationUnsubmitted).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
				WithArgs(42, test.legacyProvider, fields.SocialID, fields.SocialEmail).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
				WithArgs(42, string(test.canonical), fields.SocialID, test.normalizedEmail).
				WillReturnResult(sqlmock.NewResult(101, 1))
			expectCreatedSocialMember(mock, 42, fields)
			mock.ExpectCommit()

			user, err := repo.CreateSocialAccount(fields)
			if err != nil {
				t.Fatal(err)
			}
			if user.USRSeq != 42 {
				t.Fatalf("USRSeq = %d, want 42", user.USRSeq)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateSocialAccountSkipsCanonicalWritesWhenNotReady(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	fields := socialAccountFixture(string(model.SocialProviderKakao), "member@example.com")

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WithArgs(42, model.VerificationUnsubmitted).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, string(model.SocialProviderKakao), fields.SocialID, fields.SocialEmail).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectCreatedSocialMember(mock, 42, fields)
	mock.ExpectCommit()

	if _, err := repo.CreateSocialAccount(fields); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSocialAccountRollsBackWhenCanonicalIdentityInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnableCanonicalIdentityWrites()
	fields := socialAccountFixture(string(model.SocialProviderKakao), "member@example.com")

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO AUTH_ACCOUNT_STATE`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
		WillReturnError(errors.New("duplicate canonical identity"))
	mock.ExpectRollback()

	if _, err := repo.CreateSocialAccount(fields); err == nil {
		t.Fatal("CreateSocialAccount() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSocialAccountClassifiesDuplicateSocialIdentity(t *testing.T) {
	tests := []struct {
		name               string
		canonicalEnabled   bool
		expectSocialInsert func(sqlmock.Sqlmock)
	}{
		{
			name: "legacy social connection",
			expectSocialInsert: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
					WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'KT-subject' for key 'UK_PROVIDER_SUBJECT'"})
			},
		},
		{
			name:             "canonical social identity",
			canonicalEnabled: true,
			expectSocialInsert: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
					WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'KAKAO-subject' for key 'UQ_AUTH_IDENTITY_PROVIDER_SUBJECT'"})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			if test.canonicalEnabled {
				repo.EnableCanonicalIdentityWrites()
			}
			fields := socialAccountFixture(string(model.SocialProviderKakao), "member@example.com")

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
				WillReturnResult(sqlmock.NewResult(42, 1))
			if test.canonicalEnabled {
				mock.ExpectExec(`INSERT INTO AUTH_ACCOUNT_STATE`).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}
			mock.ExpectExec(`INSERT INTO ALUMNI_VERIFICATION`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			test.expectSocialInsert(mock)
			mock.ExpectRollback()

			_, err = repo.CreateSocialAccount(fields)
			if !errors.Is(err, ErrSocialIdentityAlreadyLinked) {
				t.Fatalf("CreateSocialAccount() error = %v, want ErrSocialIdentityAlreadyLinked", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func socialAccountFixture(provider string, email string) SocialAccountFields {
	return SocialAccountFields{
		Provider:    provider,
		SocialID:    "social-subject",
		SocialEmail: "provider-profile@example.net",
		USRID:       "social-member",
		Name:        "Social Member",
		Phone:       "01012345678",
		Email:       email,
	}
}

func expectCreatedSocialMember(mock sqlmock.Sqlmock, usrSeq int, fields SocialAccountFields) {
	mock.ExpectQuery(`FROM WEO_MEMBER`).
		WithArgs(usrSeq).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE", "USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(
			usrSeq, fields.USRID, fields.Name, "BBB", fields.Phone, fields.FN, fields.Email, "", nil,
		))
}
