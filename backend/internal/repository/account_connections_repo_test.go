package repository_test

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TestLinkSocialIdentityWritesLegacyAndCanonicalConnections(t *testing.T) {
	tests := []struct {
		name             string
		provider         model.SocialProvider
		canonical        model.IdentityProvider
		canonicalEnabled bool
	}{
		{name: "kakao legacy write", provider: model.SocialProviderKakao},
		{name: "apple canonical write", provider: model.SocialProviderApple, canonical: model.IdentityProviderApple, canonicalEnabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			if test.canonicalEnabled {
				repo.EnableCanonicalIdentityWrites()
			}
			fields := repository.SocialAccountFields{
				USRSeq:      42,
				Provider:    string(test.provider),
				SocialID:    "provider-subject",
				SocialEmail: "Member@Example.COM",
				Email:       "Member@Example.COM",
			}

			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
				WithArgs(42, string(test.provider), fields.SocialID, fields.SocialEmail).
				WillReturnResult(sqlmock.NewResult(1, 1))
			if test.canonicalEnabled {
				mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
					WithArgs(42, string(test.canonical), fields.SocialID, "member@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}
			mock.ExpectCommit()

			if err := repo.LinkSocialIdentity(fields); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLinkSocialIdentityClassifiesLegacyDuplicateForServiceRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate provider subject"})
	mock.ExpectRollback()

	err = repo.LinkSocialIdentity(repository.SocialAccountFields{
		USRSeq:   42,
		Provider: string(model.SocialProviderKakao),
		SocialID: "provider-subject",
	})
	if !errors.Is(err, repository.ErrSocialIdentityAlreadyLinked) {
		t.Fatalf("error = %v, want ErrSocialIdentityAlreadyLinked", err)
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
		t.Fatalf("error = %v, want wrapped MySQL 1062 error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkSocialIdentityRollsBackLegacyInsertWhenCanonicalIdentityIsDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnableCanonicalIdentityWrites()

	fields := repository.SocialAccountFields{
		USRSeq:      42,
		Provider:    string(model.SocialProviderKakao),
		SocialID:    "provider-subject",
		SocialEmail: "member@example.com",
		Email:       "member@example.com",
	}
	duplicateErr := &mysql.MySQLError{Number: 1062, Message: "duplicate canonical identity"}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WithArgs(42, string(model.SocialProviderKakao), fields.SocialID, fields.SocialEmail).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO AUTH_IDENTITY`).
		WithArgs(42, string(model.IdentityProviderKakao), fields.SocialID, fields.Email).
		WillReturnError(duplicateErr)
	mock.ExpectRollback()

	err = repo.LinkSocialIdentity(fields)
	if !errors.Is(err, repository.ErrSocialIdentityAlreadyLinked) {
		t.Fatalf("error = %v, want ErrSocialIdentityAlreadyLinked", err)
	}
	if !errors.Is(err, duplicateErr) {
		t.Fatalf("error = %v, want wrapped canonical insert error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAccountConnections(t *testing.T) {
	tests := []struct {
		name            string
		passwordPresent int
		wantPassword    bool
	}{
		{name: "has password", passwordPresent: 1, wantPassword: true},
		{name: "has no password", passwordPresent: 0, wantPassword: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

			mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*ORDER BY NMS_GATE`).
				WithArgs(42).
				WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP").AddRow("KT"))
			mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
				WithArgs(42).
				WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(tt.passwordPresent))

			connections, err := repo.GetAccountConnections(42)
			if err != nil {
				t.Fatal(err)
			}
			if len(connections.Providers) != 2 || connections.Providers[0] != "AP" || connections.Providers[1] != "KT" {
				t.Fatalf("providers = %#v", connections.Providers)
			}
			if connections.HasPassword != tt.wantPassword {
				t.Fatalf("hasPassword = %v, want %v", connections.HasPassword, tt.wantPassword)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetAccountConnectionsReturnsEmptyProviderSlice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`SELECT NMS_GATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(1))

	connections, err := repo.GetAccountConnections(42)
	if err != nil {
		t.Fatal(err)
	}
	if connections.Providers == nil || len(connections.Providers) != 0 {
		t.Fatalf("providers must be a non-nil empty slice: %#v", connections.Providers)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSocialConnectionLocksAndRemovesRelatedRecordsWithoutCanonicalWrites(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP").AddRow("KT"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(0))
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

	if err := repo.DeleteSocialConnection(42, "KT"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSocialConnectionRevokesCanonicalIdentityWhenWritesEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnableCanonicalIdentityWrites()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(1))
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE AUTH_IDENTITY[\s\S]*STATUS = 'REVOKED'[\s\S]*REVOKED_AT = NOW\(\)[\s\S]*ACCOUNT_ID = \?[\s\S]*PROVIDER = \?[\s\S]*STATUS = 'ACTIVE'`).
		WithArgs(42, "APPLE").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := repo.DeleteSocialConnection(42, "AP"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSocialConnectionRejectsLastLoginMethodInsideTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("KT"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(0))
	mock.ExpectRollback()

	err = repo.DeleteSocialConnection(42, "KT")
	if !errors.Is(err, repository.ErrLastLoginMethod) {
		t.Fatalf("error = %v, want repository.ErrLastLoginMethod", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestForceDeleteSocialConnectionRemovesLastLoginMethodAndRevokesCanonicalIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	repo.EnableCanonicalIdentityWrites()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NMS_GATE[\s\S]*NMS_STATUS IN \('Y', 'ACTIVE'\)[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"NMS_GATE"}).AddRow("AP"))
	mock.ExpectQuery(`COALESCE\(TRIM\(USR_PWD\), ''\) <> ''[\s\S]*FOR UPDATE`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"has_password"}).AddRow(0))
	mock.ExpectExec(`DELETE FROM WEO_MEMBER_SOCIAL`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE AUTH_IDENTITY[\s\S]*STATUS = 'REVOKED'[\s\S]*REVOKED_AT = NOW\(\)[\s\S]*ACCOUNT_ID = \?[\s\S]*PROVIDER = \?[\s\S]*STATUS = 'ACTIVE'`).
		WithArgs(42, "APPLE").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM ALUMNI_SOCIAL_CREDENTIAL`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX`).
		WithArgs(42, "AP").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := repo.ForceDeleteSocialConnection(42, "AP"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
