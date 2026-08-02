package repository

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func newPushRepositoryTest(t *testing.T) (*PushRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewPushRepository(sqlx.NewDb(db, "sqlmock")), mock, func() { _ = db.Close() }
}

func TestPushRepositoryRegistersDeviceWithAccountReassignmentUpsert(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	environment, bundleID := "production", "com.daeil.dflhsafv2"
	request := model.PushDeviceRegistration{
		Platform: "ios", DeviceToken: "token-123", Locale: "ko_KR",
		APNSEnvironment: &environment, BundleID: &bundleID,
	}
	mock.ExpectExec(`(?s)INSERT INTO ALUMNI_PUSH_DEVICE.*LAST_SEEN_AT.*CREATED_AT.*UPDATED_AT.*VALUES.*UTC_TIMESTAMP\(\).*ON DUPLICATE KEY UPDATE.*LAST_SEEN_AT = UTC_TIMESTAMP\(\).*UPDATED_AT = UTC_TIMESTAMP\(\)`).
		WithArgs(42, "ios", "token-123", "ko_KR", "production", "com.daeil.dflhsafv2").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.RegisterDevice(42, request); err != nil {
		t.Fatalf("RegisterDevice error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryRegistersAndroidWithNullAPNSFields(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectExec(`(?s)INSERT INTO ALUMNI_PUSH_DEVICE.*LAST_SEEN_AT.*CREATED_AT.*UPDATED_AT.*VALUES.*UTC_TIMESTAMP\(\).*ON DUPLICATE KEY UPDATE.*LAST_SEEN_AT = UTC_TIMESTAMP\(\).*UPDATED_AT = UTC_TIMESTAMP\(\)`).
		WithArgs(42, "android", "token-123", "ko-KR", nil, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.RegisterDevice(42, model.PushDeviceRegistration{
		Platform: "android", DeviceToken: "token-123", Locale: "ko-KR",
	}); err != nil {
		t.Fatalf("RegisterDevice error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryUnregistersOnlyCurrentOwnersMatchingToken(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM ALUMNI_PUSH_DEVICE`)).
		WithArgs(42, "token-123").
		WillReturnResult(sqlmock.NewResult(0, 2))

	if err := repo.UnregisterDevice(42, "token-123"); err != nil {
		t.Fatalf("UnregisterDevice error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryReturnsNilForMissingPreferences(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT MESSAGE_ENABLED, MESSAGE_PREVIEW_ENABLED`)).
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)

	preferences, err := repo.GetPreferences(42)
	if err != nil || preferences != nil {
		t.Fatalf("preferences = %#v, error = %v", preferences, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryMapsStoredPreferenceFlags(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT MESSAGE_ENABLED, MESSAGE_PREVIEW_ENABLED`)).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"MESSAGE_ENABLED", "MESSAGE_PREVIEW_ENABLED"}).AddRow("Y", "N"))

	preferences, err := repo.GetPreferences(42)
	if err != nil {
		t.Fatal(err)
	}
	if preferences == nil || !preferences.MessageEnabled || preferences.MessagePreviewEnabled {
		t.Fatalf("preferences = %#v", preferences)
	}
}

func TestPushRepositoryUpsertsCanonicalPreferenceFlags(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectExec(`(?s)INSERT INTO ALUMNI_PUSH_PREFERENCE.*CREATED_AT.*UPDATED_AT.*VALUES.*UTC_TIMESTAMP\(\).*ON DUPLICATE KEY UPDATE.*UPDATED_AT = UTC_TIMESTAMP\(\)`).
		WithArgs(42, "N", "Y").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertPreferences(42, model.PushPreferences{MessageEnabled: false, MessagePreviewEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryListsEveryDeviceForRecipient(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT PLATFORM, DEVICE_TOKEN, APNS_ENVIRONMENT, BUNDLE_ID`)).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"PLATFORM", "DEVICE_TOKEN", "APNS_ENVIRONMENT", "BUNDLE_ID"}).
			AddRow("android", "android-token", nil, nil).
			AddRow("ios", "ios-token", "sandbox", "com.daeil.dflhsafv2"))

	targets, err := repo.ListDevices(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 || targets[0].Platform != "android" || targets[1].APNSEnvironment != "sandbox" {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestPushRepositoryDeletesExactInvalidPlatformToken(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM ALUMNI_PUSH_DEVICE WHERE PLATFORM = ? AND DEVICE_TOKEN = ?`)).
		WithArgs("ios", "invalid-token").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.DeleteDevice("ios", "invalid-token"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
