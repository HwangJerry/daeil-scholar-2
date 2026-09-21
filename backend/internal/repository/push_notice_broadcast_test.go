package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// A page returns members, not device rows, so a member can never straddle a
// page boundary and be enqueued twice.
func TestPushRepositoryPagesNoticeRecipientsByMember(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectQuery(`(?s)SELECT DISTINCT d\.USR_SEQ.*FROM ALUMNI_MOBILE_DEVICE_TOKEN d.*ORDER BY d\.USR_SEQ ASC.*LIMIT \?`).
		WithArgs(70, 2).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(71).AddRow(72))

	seqs, err := repo.ListNoticeRecipientSeqs(70, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(seqs) != 2 || seqs[0] != 71 || seqs[1] != 72 {
		t.Fatalf("seqs = %#v", seqs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Members who turned notices off are excluded in SQL, and members with no
// preference row at all are treated as opted in.
func TestPushRepositoryFiltersNoticeRecipientsByOptOutWithDefaultOn(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	mock.ExpectQuery(`(?s)LEFT JOIN ALUMNI_PUSH_PREFERENCE p ON p\.USR_SEQ = d\.USR_SEQ.*WHERE d\.STATUS = 'ACTIVE'.*COALESCE\(p\.NOTICE_ENABLED, 'Y'\) = 'Y'.*d\.USR_SEQ > \?`).
		WithArgs(0, 200).
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(5))

	seqs, err := repo.ListNoticeRecipientSeqs(0, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(seqs) != 1 || seqs[0] != 5 {
		t.Fatalf("seqs = %#v", seqs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryRejectsNonPositiveNoticePageSizeWithoutQuerying(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()

	seqs, err := repo.ListNoticeRecipientSeqs(0, 0)
	if err != nil || len(seqs) != 0 {
		t.Fatalf("seqs = %#v, error = %v", seqs, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPushRepositoryReportsDeletedNoticeAsUnpublished(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		count int
		want  bool
	}{
		{name: "published", count: 1, want: true},
		{name: "deleted", count: 0, want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repo, mock, cleanup := newPushRepositoryTest(t)
			defer cleanup()
			mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM WEO_BOARDBBS.*WHERE SEQ = \? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'`).
				WithArgs(501).
				WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(testCase.count))

			published, err := repo.NoticeStillPublished(501)
			if err != nil {
				t.Fatal(err)
			}
			if published != testCase.want {
				t.Fatalf("published = %v, want %v", published, testCase.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
