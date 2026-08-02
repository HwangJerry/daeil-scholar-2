package service

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestSearchAlumniUsesApprovedVerificationAndCanonicalResponse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	alumniService := NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)

	mock.ExpectQuery(`JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectQuery(`JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*ORDER BY m.USR_NAME ASC, m.USR_SEQ ASC`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_NAME", "USR_PHOTO", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "AJC_NAME", "USR_POSITION",
		}).AddRow(202, "예시 동문", nil, 2004, "18", "영어", "교육", "교사"))

	response, err := alumniService.Search(model.AlumniSearchParams{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, required := range []string{
		`"userSeq":202`, `"name":"예시 동문"`, `"photoUrl":null`,
		`"cohort":"18"`, `"department":"영어"`, `"jobCategory":"교육"`, `"jobRole":"교사"`,
		`"page":1`, `"size":20`, `"totalCount":1`, `"totalPages":1`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("body = %s, missing %s", body, required)
		}
	}
	for _, forbidden := range []string{"fmSeq", "fmName", "fmFn", "fmDept", "weeklyCount", "phone", "email", "bizCard", "tags"} {
		if strings.Contains(body, `"`+forbidden+`"`) {
			t.Fatalf("body = %s, contains forbidden field %s", body, forbidden)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSearchAlumniAppliesCanonicalCompositeFiltersAndPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	alumniService := NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)

	filterArgs := []driver.Value{"%홍길동%", 2004, "18", "영어", 3, "교사"}
	mock.ExpectQuery(`m.USR_NAME LIKE \?[\s\S]*v.GRADUATION_YEAR = \?[\s\S]*v.COHORT = \?[\s\S]*v.DEPARTMENT = \?[\s\S]*m.USR_JOB_CAT = \?[\s\S]*m.USR_POSITION = \?`).
		WithArgs(filterArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(51))
	selectArgs := append(filterArgs, 50, 50)
	mock.ExpectQuery(`m.USR_NAME LIKE \?[\s\S]*ORDER BY m.USR_NAME ASC, m.USR_SEQ ASC[\s\S]*LIMIT \? OFFSET \?`).
		WithArgs(selectArgs...).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_NAME", "USR_PHOTO", "GRADUATION_YEAR", "COHORT", "DEPARTMENT", "AJC_NAME", "USR_POSITION",
		}))

	response, err := alumniService.Search(model.AlumniSearchParams{
		Name:           "  홍길동  ",
		GraduationYear: 2004,
		Cohort:         " 18 ",
		Department:     " 영어 ",
		JobCategory:    3,
		JobRole:        " 교사 ",
		Page:           2,
		Size:           60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Page != 2 || response.Size != 50 || response.TotalCount != 51 || response.TotalPages != 2 {
		t.Fatalf("pagination = %#v", response)
	}
	if response.Items == nil || len(response.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty array", response.Items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
