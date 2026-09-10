package service

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestGetAlumniDetailAppliesIndependentContactDisclosure(t *testing.T) {
	tests := []struct {
		name        string
		phone       any
		phonePublic any
		email       any
		emailPublic any
		blockedByMe bool
		wantPhone   bool
		wantEmail   bool
	}{
		{
			name:  "public phone and private email",
			phone: "01000000000", phonePublic: "Y",
			email: "private@example.com", emailPublic: "N",
			blockedByMe: true, wantPhone: true,
		},
		{
			name:  "private phone and public email",
			phone: "01011111111", phonePublic: "N",
			email: "public@example.com", emailPublic: "Y",
			wantEmail: true,
		},
		{
			name:  "unknown flags do not expose values",
			phone: "01099999999", phonePublic: nil,
			email: "hidden@example.com", emailPublic: "",
		},
		{
			name:  "public flags with empty values",
			phone: "", phonePublic: "Y",
			email: nil, emailPublic: "Y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			alumniService := NewAlumniService(repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock")), nil)

			mock.ExpectQuery(`JOIN ALUMNI_VERIFICATION v[\s\S]*v.STATUS = 'approved'[\s\S]*m.USR_SEQ = \?`).
				WithArgs(101, 202).
				WillReturnRows(sqlmock.NewRows([]string{
					"USR_SEQ", "USR_NAME", "USR_PHOTO", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
					"AJC_NAME", "USR_POSITION", "USR_PHONE", "USR_EMAIL", "USR_PHONE_PUBLIC",
					"USR_EMAIL_PUBLIC", "BLOCKED_BY_ME", "USR_BIZ_NAME", "USR_BIZ_ADDR", "USR_BIZ_DESC", "USR_BIZ_CARD",
				}).AddRow(
					202, "예시 동문", nil, 2004, "18", "영어",
					"교육", "교사", tt.phone, tt.email, tt.phonePublic, tt.emailPublic, tt.blockedByMe, "대일", "서울", "소개글", "/uploads/card.jpg",
				))

			mock.ExpectQuery(`SELECT AUT_TAG FROM ALUMNI_USER_TAG WHERE USR_SEQ = \?`).WithArgs(202).
				WillReturnRows(sqlmock.NewRows([]string{"AUT_TAG"}).AddRow("교육").AddRow("멘토링"))

			detail, err := alumniService.GetDetail(101, 202)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(detail)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(encoded, &payload); err != nil {
				t.Fatal(err)
			}

			baseKeys := map[string]bool{
				"userSeq": true, "name": true, "photoUrl": true, "cohort": true,
				"department": true, "jobCategory": true, "jobRole": true, "blockState": true,
				"graduationYear": true, "bizName": true, "bizAddr": true, "bizDesc": true,
				"bizCardUrl": true, "tags": true, "phonePublic": true, "emailPublic": true,
			}
			if tt.wantPhone {
				baseKeys["phone"] = true
			}
			if tt.wantEmail {
				baseKeys["email"] = true
			}
			if len(payload) != len(baseKeys) {
				t.Fatalf("payload keys = %v, want exactly %v", payload, baseKeys)
			}
			for key := range payload {
				if !baseKeys[key] {
					t.Fatalf("unexpected detail field %q in %s", key, encoded)
				}
			}
			if payload["phonePublic"] != (tt.phonePublic == "Y") || payload["emailPublic"] != (tt.emailPublic == "Y") {
				t.Fatalf("independent disclosure flags: %s", encoded)
			}
			if payload["graduationYear"] != float64(2004) || payload["bizName"] != "대일" || payload["bizAddr"] != "서울" || payload["bizDesc"] != "소개글" || payload["bizCardUrl"] != "/uploads/card.jpg" || len(detail.Tags) != 2 {
				t.Fatalf("missing full profile: %s", encoded)
			}
			blockState, ok := payload["blockState"].(map[string]any)
			if !ok || len(blockState) != 1 || blockState["blockedByMe"] != tt.blockedByMe {
				t.Fatalf("blockState = %#v", payload["blockState"])
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
