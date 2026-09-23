package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
					"USR_SEQ", "USR_NAME", "USR_PHOTO", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
					"AJC_NAME", "USR_POSITION", "USR_BIZ_NAME", "USR_BIZ_CARD",
				}).AddRow(202, "예시 동문", "/files/profile/example.jpg", 2004, "18", "영어", "교육", "교사", tt.bizName, tt.bizCard))
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
