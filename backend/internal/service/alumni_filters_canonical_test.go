package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestGetAlumniFiltersReturnsCanonicalApprovedOptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	alumniService := NewAlumniService(
		repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")),
		cache.New(time.Minute, time.Minute),
	)

	mock.ExpectQuery(`SELECT DISTINCT v.GRADUATION_YEAR[\s\S]*v.STATUS = 'approved'[\s\S]*ORDER BY v.GRADUATION_YEAR DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"GRADUATION_YEAR"}).AddRow(2004).AddRow(2003))
	mock.ExpectQuery(`SELECT DISTINCT v.COHORT[\s\S]*v.STATUS = 'approved'[\s\S]*REGEXP`).
		WillReturnRows(sqlmock.NewRows([]string{"COHORT"}).AddRow("18").AddRow("19"))
	mock.ExpectQuery(`SELECT DISTINCT v.DEPARTMENT[\s\S]*v.STATUS = 'approved'[\s\S]*ORDER BY v.DEPARTMENT ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"DEPARTMENT"}).AddRow("독일어").AddRow("영어"))
	mock.ExpectQuery(`FROM ALUMNI_JOB_CATEGORY[\s\S]*OPEN_YN = 'Y'[\s\S]*ORDER BY AJC_INDX ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"AJC_SEQ", "AJC_NAME"}).AddRow(3, "교육"))
	mock.ExpectQuery(`SELECT DISTINCT m.USR_POSITION[\s\S]*JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*ORDER BY m.USR_POSITION ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"USR_POSITION"}).AddRow("교사").AddRow("연구원"))

	filters, err := alumniService.GetFilters()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(filters)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, required := range []string{
		`"graduationYears":[2004,2003]`,
		`"cohorts":["18","19"]`,
		`"departments":["독일어","영어"]`,
		`"jobCategories":[{"seq":3,"name":"교육"}]`,
		`"jobRoles":["교사","연구원"]`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("body = %s, missing %s", body, required)
		}
	}
	for _, forbidden := range []string{"fnList", "deptList"} {
		if strings.Contains(body, `"`+forbidden+`"`) {
			t.Fatalf("body = %s, contains legacy field %s", body, forbidden)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
