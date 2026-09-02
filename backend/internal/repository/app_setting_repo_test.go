// app_setting_repo_test.go — SQL contract tests for application settings persistence.
package repository_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestAppSettingRepositoryListsAllSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAppSettingRepository(sqlx.NewDb(db, "sqlmock"))
	updatedAt := time.Date(2026, 9, 2, 12, 0, 0, 0, time.Local)

	mock.ExpectQuery(`(?s)SELECT AS_KEY, AS_VALUE, AS_DESCRIPTION, AS_PUBLIC, UPDATED_AT, UPDATED_BY.*FROM app_settings.*ORDER BY AS_KEY ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"AS_KEY", "AS_VALUE", "AS_DESCRIPTION", "AS_PUBLIC", "UPDATED_AT", "UPDATED_BY",
		}).AddRow("kakao_open_chat_url", "https://example.com/chat", "문의 URL", "Y", updatedAt, 7))

	settings, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(settings) != 1 || settings[0].Key != "kakao_open_chat_url" || settings[0].UpdatedBy == nil || *settings[0].UpdatedBy != 7 {
		t.Fatalf("ListAll() = %#v", settings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppSettingRepositoryListsOnlyPublicSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := repository.NewAppSettingRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectQuery(`(?s)FROM app_settings.*WHERE AS_PUBLIC = 'Y'.*ORDER BY AS_KEY ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"AS_KEY", "AS_VALUE", "AS_DESCRIPTION", "AS_PUBLIC", "UPDATED_AT", "UPDATED_BY",
		}))

	settings, err := repo.ListPublic()
	if err != nil {
		t.Fatalf("ListPublic() error = %v", err)
	}
	if settings == nil || len(settings) != 0 {
		t.Fatalf("ListPublic() = %#v, want empty non-nil slice", settings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppSettingRepositoryUpdateDistinguishesUnchangedFromMissing(t *testing.T) {
	tests := []struct {
		name       string
		exists     int
		wantExists bool
	}{
		{name: "unchanged existing row", exists: 1, wantExists: true},
		{name: "missing row", exists: 0, wantExists: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := repository.NewAppSettingRepository(sqlx.NewDb(db, "sqlmock"))

			mock.ExpectExec(`(?s)UPDATE app_settings.*SET AS_VALUE = \?, UPDATED_AT = NOW\(\), UPDATED_BY = \?.*WHERE AS_KEY = \?`).
				WithArgs("new-value", 7, "setting-key").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM app_settings.*WHERE AS_KEY = \?`).
				WithArgs("setting-key").
				WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(test.exists))

			exists, err := repo.UpdateValue("setting-key", "new-value", 7)
			if err != nil {
				t.Fatalf("UpdateValue() error = %v", err)
			}
			if exists != test.wantExists {
				t.Fatalf("UpdateValue() exists = %v, want %v", exists, test.wantExists)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
