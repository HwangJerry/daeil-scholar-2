package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestLegacyLogoutRevocationAlsoRevokesCanonicalRotationState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`SET MRT_REVOKED_AT = COALESCE\(MRT_REVOKED_AT, NOW\(\)\),\s+REVOKED_AT = COALESCE\(REVOKED_AT, NOW\(\)\)`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	if err := repo.RevokeMobileRefreshTokensByUser(42); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyRefreshConsumeRejectsCanonicalConsumedToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectExec(`AND MRT_REVOKED_AT IS NULL AND REVOKED_AT IS NULL AND CONSUMED_AT IS NULL AND EXPIRES_AT > NOW\(\)`).
		WithArgs("old-jti", 42).
		WillReturnResult(sqlmock.NewResult(0, 0))
	consumed, err := repo.RevokeMobileRefreshToken(42, "old-jti")
	if err != nil {
		t.Fatal(err)
	}
	if consumed {
		t.Fatal("canonical-consumed legacy token was consumed again")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
