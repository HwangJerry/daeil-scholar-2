package repository

import (
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type nullStringArg struct {
	value string
	valid bool
}

func (a nullStringArg) Match(value driver.Value) bool {
	if !a.valid {
		return value == nil
	}
	got, ok := value.(string)
	return ok && got == a.value
}

func TestMobileDeviceTokenRepositoryUpsertTokenStoresAndroidAPNsMetadataAsNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec("INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN").
		WithArgs(10, "android", "fcm-token-1", nullStringArg{}, nullStringArg{}, "ko-KR").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpsertToken(10, model.PushDeviceRegistrationRequest{
		Platform:        "android",
		DeviceToken:     "fcm-token-1",
		APNsEnvironment: "sandbox",
		BundleID:        "com.daeil.dflhsafv2",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMobileDeviceTokenRepositoryUpsertTokenPreservesIOSAPNsMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	mock.ExpectExec("INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN").
		WithArgs(
			10,
			"ios",
			"apns-token-1",
			nullStringArg{value: "sandbox", valid: true},
			nullStringArg{value: "com.daeil.dflhsafv2", valid: true},
			"ko-KR",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpsertToken(10, model.PushDeviceRegistrationRequest{
		Platform:        "ios",
		DeviceToken:     "apns-token-1",
		APNsEnvironment: "sandbox",
		BundleID:        "com.daeil.dflhsafv2",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMobileDeviceTokenRepositoryGetActiveTokensByUserReadsAndroidAPNsEnvironmentAsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	rows := sqlmock.NewRows([]string{
		"MDT_SEQ",
		"USR_SEQ",
		"PLATFORM",
		"DEVICE_TOKEN",
		"APNS_ENVIRONMENT",
		"BUNDLE_ID",
	}).AddRow(1, 10, "android", "fcm-token-1", "", "")

	mock.ExpectQuery(`(?s)CASE\s+WHEN PLATFORM = 'ios' THEN COALESCE\(APNS_ENVIRONMENT, 'production'\)\s+ELSE ''\s+END AS APNS_ENVIRONMENT.*CASE\s+WHEN PLATFORM = 'ios' THEN COALESCE\(BUNDLE_ID, ''\)\s+ELSE ''\s+END AS BUNDLE_ID.*FROM ALUMNI_MOBILE_DEVICE_TOKEN`).
		WithArgs(10).
		WillReturnRows(rows)

	tokens, err := repo.GetActiveTokensByUser(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one token, got %d", len(tokens))
	}
	if tokens[0].APNsEnvironment != "" {
		t.Fatalf("expected android APNs environment to be empty, got %#v", tokens[0])
	}
	if tokens[0].BundleID != "" {
		t.Fatalf("expected android bundle ID to be empty, got %#v", tokens[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMobileDeviceTokenRepositoryGetActiveTokensForBroadcastReadsAndroidMetadataAsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	rows := sqlmock.NewRows([]string{
		"MDT_SEQ",
		"USR_SEQ",
		"PLATFORM",
		"DEVICE_TOKEN",
		"APNS_ENVIRONMENT",
		"BUNDLE_ID",
	}).AddRow(1, 10, "android", "fcm-token-1", "", "")

	mock.ExpectQuery(`(?s)CASE\s+WHEN dt\.PLATFORM = 'ios' THEN COALESCE\(dt\.APNS_ENVIRONMENT, 'production'\)\s+ELSE ''\s+END AS APNS_ENVIRONMENT.*CASE\s+WHEN dt\.PLATFORM = 'ios' THEN COALESCE\(dt\.BUNDLE_ID, ''\)\s+ELSE ''\s+END AS BUNDLE_ID.*FROM ALUMNI_MOBILE_DEVICE_TOKEN dt`).
		WithArgs(99).
		WillReturnRows(rows)

	tokens, err := repo.GetActiveTokensForBroadcast(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one token, got %d", len(tokens))
	}
	if tokens[0].APNsEnvironment != "" {
		t.Fatalf("expected android APNs environment to be empty, got %#v", tokens[0])
	}
	if tokens[0].BundleID != "" {
		t.Fatalf("expected android bundle ID to be empty, got %#v", tokens[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMobileDeviceTokenRepositoryGetActiveTokensForBroadcastReadsIOSNullAPNsEnvironmentAsProduction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	rows := sqlmock.NewRows([]string{
		"MDT_SEQ",
		"USR_SEQ",
		"PLATFORM",
		"DEVICE_TOKEN",
		"APNS_ENVIRONMENT",
		"BUNDLE_ID",
	}).AddRow(2, 11, "ios", "apns-token-1", "production", "com.daeil.dflhsafv2")

	mock.ExpectQuery(`(?s)CASE\s+WHEN dt\.PLATFORM = 'ios' THEN COALESCE\(dt\.APNS_ENVIRONMENT, 'production'\)\s+ELSE ''\s+END AS APNS_ENVIRONMENT.*FROM ALUMNI_MOBILE_DEVICE_TOKEN dt`).
		WithArgs(10).
		WillReturnRows(rows)

	tokens, err := repo.GetActiveTokensForBroadcast(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one token, got %d", len(tokens))
	}
	if tokens[0].APNsEnvironment != "production" {
		t.Fatalf("expected iOS APNs environment fallback to production, got %#v", tokens[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMobileDeviceTokenRepositoryGetActiveTokensByUserReadsIOSNullAPNsEnvironmentAsProduction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewMobileDeviceTokenRepository(sqlx.NewDb(db, "sqlmock"))
	rows := sqlmock.NewRows([]string{
		"MDT_SEQ",
		"USR_SEQ",
		"PLATFORM",
		"DEVICE_TOKEN",
		"APNS_ENVIRONMENT",
		"BUNDLE_ID",
	}).AddRow(2, 11, "ios", "apns-token-1", "production", "com.daeil.dflhsafv2")

	mock.ExpectQuery(`(?s)CASE\s+WHEN PLATFORM = 'ios' THEN COALESCE\(APNS_ENVIRONMENT, 'production'\)\s+ELSE ''\s+END AS APNS_ENVIRONMENT.*FROM ALUMNI_MOBILE_DEVICE_TOKEN`).
		WithArgs(11).
		WillReturnRows(rows)

	tokens, err := repo.GetActiveTokensByUser(11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected one token, got %d", len(tokens))
	}
	if tokens[0].APNsEnvironment != "production" {
		t.Fatalf("expected iOS APNs environment fallback to production, got %#v", tokens[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
