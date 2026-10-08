package handler

import (
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
)

func TestSearchAlumniReturnsBusinessFieldsWithCanonicalContract(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "contracts", "fixtures", "alumni-search.json"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		bizName     any
		bizCard     any
		useFixture  bool
		wantBizName string
		wantBizCard any
	}{
		{name: "canonical fixture", bizName: "대일", bizCard: "/uploads/card.jpg", useFixture: true},
		{name: "SQL NULL business fields"},
		{name: "empty business fields", bizName: "", bizCard: ""},
		{name: "company without card", bizName: "대일", wantBizName: "대일"},
		{
			name: "legacy card without company", bizCard: "/files/card/legacy.jpg",
			wantBizCard: "/files/card/legacy.jpg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(`SELECT COUNT\(\*\)`).
				WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
			mock.ExpectQuery(`m.USR_BIZ_NAME, m.USR_BIZ_CARD[\s\S]*LIMIT \? OFFSET \?`).
				WithArgs(20, 0).
				WillReturnRows(sqlmock.NewRows([]string{
					"USR_SEQ", "USR_NAME", "USR_PHOTO", "MESSAGE_ALLOWED", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
					"AJC_NAME", "USR_POSITION", "USR_BIZ_NAME", "USR_BIZ_CARD",
				}).AddRow(202, "예시 동문", "/files/profile/example.jpg", true, 2004, "18", "영어", "교육", "교사", tt.bizName, tt.bizCard))
			alumniService := service.NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)
			recorder := httptest.NewRecorder()
			NewAlumniHandler(alumniService).Search(recorder, httptest.NewRequest(http.MethodGet, "/api/alumni", nil))

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			var got, want map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(fixture, &want); err != nil {
				t.Fatal(err)
			}
			if !tt.useFixture {
				items, ok := want["items"].([]any)
				if !ok || len(items) != 1 {
					t.Fatal("canonical fixture must have one alumni item")
				}
				item, ok := items[0].(map[string]any)
				if !ok {
					t.Fatal("canonical alumni item must be an object")
				}
				item["bizName"] = tt.wantBizName
				item["bizCardUrl"] = tt.wantBizCard
			}
			// Compare the entire response so missing null fields and extra profile fields fail.
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("response = %#v, want %#v", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSearchAlumniMultiSelectionAndLegacyRequests(t *testing.T) {
	cases := []struct {
		name, query, where string
		args               []driver.Value
	}{
		{"plural", "cohorts=30,31,30&departments=%20영어%20,스페인어&jobCategories=2,3,2,0&name=김&page=2&size=2",
			`m.USR_NAME LIKE \?[\s\S]*v.COHORT IN \(\?,\?\)[\s\S]*v.DEPARTMENT IN \(\?,\?\)[\s\S]*m.USR_JOB_CAT IN \(\?,\?\)`,
			[]driver.Value{"%김%", "30", "31", "영어", "스페인어", 2, 3}},
		{"repeated legacy", "cohort=30&cohort=31&department=영어&department=스페인어&jobCategory=2&jobCategory=3&name=김&page=2&size=2",
			`m.USR_NAME LIKE \?[\s\S]*v.COHORT IN \(\?,\?\)[\s\S]*v.DEPARTMENT IN \(\?,\?\)[\s\S]*m.USR_JOB_CAT IN \(\?,\?\)`,
			[]driver.Value{"%김%", "30", "31", "영어", "스페인어", 2, 3}},
		{"single legacy", "cohort=30&department=영어&jobCategory=2&name=김&page=2&size=2",
			`m.USR_NAME LIKE \?[\s\S]*v.COHORT = \?[\s\S]*v.DEPARTMENT = \?[\s\S]*m.USR_JOB_CAT = \?`,
			[]driver.Value{"%김%", "30", "영어", 2}},
		{"all departments", "cohorts=30,31&departments=&jobCategories=2,3&name=김&page=2&size=2",
			`m.USR_NAME LIKE \?[\s\S]*v.COHORT IN \(\?,\?\)[\s\S]*m.USR_JOB_CAT IN \(\?,\?\)`,
			[]driver.Value{"%김%", "30", "31", 2, 3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(`SELECT COUNT\(\*\)[\s\S]*` + tc.where).WithArgs(tc.args...).
				WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(5))
			pageArgs := append(append([]driver.Value{}, tc.args...), 2, 2)
			mock.ExpectQuery(`SELECT m.USR_SEQ[\s\S]*` + tc.where + `[\s\S]*LIMIT \? OFFSET \?`).WithArgs(pageArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "USR_NAME", "COHORT", "DEPARTMENT", "AJC_NAME"}).
					AddRow(101, "김동문", "31", "스페인어", "교육"))
			h := NewAlumniHandler(service.NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil))
			recorder := httptest.NewRecorder()
			h.Search(recorder, httptest.NewRequest(http.MethodGet, "/api/alumni?"+tc.query, nil))
			if recorder.Code != 200 {
				t.Fatalf("%d %s", recorder.Code, recorder.Body.String())
			}
			var response model.AlumniSearchResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.TotalCount != 5 || response.TotalPages != 3 || response.Page != 2 || len(response.Items) != 1 {
				t.Fatalf("unexpected paginated response: %#v", response)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
