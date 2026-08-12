package service

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestGetAlumniWidgetUsesApprovedVerificationSource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	alumniService := NewAlumniService(
		repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")),
		cache.New(time.Minute, time.Minute),
	)

	mock.ExpectQuery(`COUNT\(\*\)[\s\S]*JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_STATUS IN \('CCC','ZZZ'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectQuery(`SELECT m.USR_NAME[\s\S]*JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_STATUS IN \('CCC','ZZZ'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"USR_NAME"}).AddRow("예시 동문"))

	response, err := alumniService.GetWidgetPreview()
	if err != nil {
		t.Fatal(err)
	}
	if response.TotalCount != 1 || len(response.Items) != 1 || response.Items[0].FmName != "예시 동문" {
		t.Fatalf("response = %#v", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
