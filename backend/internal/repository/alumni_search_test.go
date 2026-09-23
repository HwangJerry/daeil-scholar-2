package repository_test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestAlumniSearchReadsBusinessFieldsInPageQuery(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name        string
		bizName     any
		bizCard     any
		wantBizName sql.NullString
		wantBizCard sql.NullString
	}{
		{
			name: "uploaded card", bizName: "대일", bizCard: "/uploads/card.jpg",
			wantBizName: sql.NullString{String: "대일", Valid: true},
			wantBizCard: sql.NullString{String: "/uploads/card.jpg", Valid: true},
		},
		{
			name: "legacy card without company", bizCard: "/files/card/legacy.jpg",
			wantBizCard: sql.NullString{String: "/files/card/legacy.jpg", Valid: true},
		},
		{name: "SQL NULL business fields"},
		{
			name: "empty business fields", bizName: "", bizCard: "",
			wantBizName: sql.NullString{Valid: true},
			wantBizCard: sql.NullString{Valid: true},
		},
	}
	rows := sqlmock.NewRows([]string{
		"USR_SEQ", "USR_NAME", "USR_PHOTO", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
		"AJC_NAME", "USR_POSITION", "USR_BIZ_NAME", "USR_BIZ_CARD",
	})
	for i, tt := range tests {
		rows.AddRow(202+i, "예시 동문", nil, 2004, "18", "영어", "교육", "교사", tt.bizName, tt.bizCard)
	}

	// Only the existing count and one page query are expected, regardless of row count.
	mock.ExpectQuery(`
		SELECT COUNT(*) FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_STATUS IN ('CCC','ZZZ')
		  AND m.USR_SEQ > 0
	`).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(len(tests)))
	mock.ExpectQuery(`
		SELECT m.USR_SEQ, m.USR_NAME, m.USR_PHOTO,
			v.GRADUATION_YEAR, v.COHORT, v.DEPARTMENT,
			jc.AJC_NAME, m.USR_POSITION, m.USR_BIZ_NAME, m.USR_BIZ_CARD
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_STATUS IN ('CCC','ZZZ')
		  AND m.USR_SEQ > 0
		ORDER BY m.USR_NAME ASC, m.USR_SEQ ASC
		LIMIT ? OFFSET ?
	`).WithArgs(20, 0).WillReturnRows(rows)

	repo := repository.NewAlumniRepository(sqlx.NewDb(db, "sqlmock"))
	records, total, err := repo.Search(model.AlumniSearchParams{})
	if err != nil {
		t.Fatal(err)
	}
	if total != len(tests) || len(records) != len(tests) {
		t.Fatalf("total = %d, rows = %d, want %d", total, len(records), len(tests))
	}
	for i, tt := range tests {
		record := records[i]
		if record.USRSeq != 202+i || record.USRBizName != tt.wantBizName || record.USRBizCard != tt.wantBizCard {
			t.Errorf("%s: business fields = %d/%#v/%#v, want %d/%#v/%#v", tt.name,
				record.USRSeq, record.USRBizName, record.USRBizCard, 202+i, tt.wantBizName, tt.wantBizCard)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
