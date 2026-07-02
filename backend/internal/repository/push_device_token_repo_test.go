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
		Platform:    "android",
		DeviceToken: "fcm-token-1",
		Locale:      "ko-KR",
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
			nullStringArg{value: "kr.dflh.saf.debug", valid: true},
			"ko-KR",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpsertToken(10, model.PushDeviceRegistrationRequest{
		Platform:        "ios",
		DeviceToken:     "apns-token-1",
		APNsEnvironment: "sandbox",
		BundleID:        "kr.dflh.saf.debug",
		Locale:          "ko-KR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
