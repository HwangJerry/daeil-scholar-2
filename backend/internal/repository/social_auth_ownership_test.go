package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestSocialIdentityOwnershipUsesCanonicalProviderAndLifecycleOnly(t *testing.T) {
	for _, tc := range []struct {
		provider  model.SocialProvider
		canonical string
	}{{model.SocialProviderKakao, "KAKAO"}, {model.SocialProviderApple, "APPLE"}} {
		t.Run(tc.canonical, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			repo.EnableCanonicalIdentityWrites()
			mock.ExpectQuery(`SELECT i.STATUS AS IDENTITY_STATUS, COALESCE\(m.USR_STATUS, ''\) AS USR_STATUS\s+FROM AUTH_IDENTITY i\s+LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = i.ACCOUNT_ID\s+WHERE i.PROVIDER = \? AND i.SUBJECT_KEY = \?`).WithArgs(tc.canonical, "synthetic-subject").
				WillReturnRows(sqlmock.NewRows([]string{"IDENTITY_STATUS", "USR_STATUS"}).AddRow("REVOKED", "CCC"))
			ownership, err := repo.FindSocialIdentityOwnership(context.Background(), tc.provider, "synthetic-subject")
			if err != nil || ownership == nil || ownership.IdentityStatus != "REVOKED" || ownership.MemberStatus != "CCC" {
				t.Fatalf("ownership lookup: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSocialIdentityOwnershipLegacyRolloutAndMissingIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	if result, err := repo.FindSocialIdentityOwnership(context.Background(), model.SocialProviderKakao, "synthetic-subject"); err != nil || result != nil {
		t.Fatal("legacy rollout queried canonical ownership")
	}
	repo.EnableCanonicalIdentityWrites()
	mock.ExpectQuery(`SELECT i.STATUS AS IDENTITY_STATUS`).WithArgs("KAKAO", "synthetic-subject").WillReturnError(sql.ErrNoRows)
	if result, err := repo.FindSocialIdentityOwnership(context.Background(), model.SocialProviderKakao, "synthetic-subject"); err != nil || result != nil {
		t.Fatal("missing canonical identity was not treated as unowned")
	}
	if _, err := repo.FindSocialIdentityOwnership(context.Background(), model.SocialProvider("unknown"), "synthetic-subject"); !errors.Is(err, model.ErrUnsupportedIdentityProvider) {
		t.Fatal("unsupported provider was queried")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
