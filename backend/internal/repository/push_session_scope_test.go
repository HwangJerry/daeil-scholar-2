package repository

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
)

func TestPushSessionRegistrationRequiresLiveLockBeforeUpsert(t *testing.T) {
	for _, failure := range []string{"missing", "query", "write", "commit", "success"} {
		t.Run(failure, func(t *testing.T) {
			repo, mock, cleanup := newPushRepositoryTest(t)
			defer cleanup()
			sid := strings.Repeat("a", 32)
			mock.ExpectBegin()
			q := mock.ExpectQuery(`SELECT MRT_JTI FROM ALUMNI_MOBILE_REFRESH_TOKEN`).WithArgs(42, sid)
			switch failure {
			case "missing":
				q.WillReturnError(sql.ErrNoRows)
			case "query":
				q.WillReturnError(errors.New("query failure"))
			default:
				q.WillReturnRows(sqlmock.NewRows([]string{"MRT_JTI"}).AddRow(strings.Repeat("b", 32)))
			}
			if failure == "missing" || failure == "query" {
				mock.ExpectRollback()
			} else {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM ALUMNI_MOBILE_REFRESH_TOKEN.*FOR UPDATE`).WithArgs(strings.Repeat("b", 32), 42, sid).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
				e := mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_DEVICE_TOKEN`).WithArgs(42, "android", "synthetic", "ko-KR", nil, nil, sid)
				if failure == "write" {
					e.WillReturnError(errors.New("write failure"))
					mock.ExpectRollback()
				} else {
					e.WillReturnResult(sqlmock.NewResult(1, 1))
					if failure == "commit" {
						mock.ExpectCommit().WillReturnError(errors.New("commit failure"))
					} else {
						mock.ExpectCommit()
					}
				}
			}
			err := repo.RegisterDevice(42, model.PushDeviceRegistration{Platform: "android", DeviceToken: "synthetic", Locale: "ko-KR", SessionID: sid})
			if failure == "success" && err != nil {
				t.Fatal(err)
			}
			if failure != "success" && err == nil {
				t.Fatal("failure suppressed")
			}
			if failure == "missing" && !errors.Is(err, ErrRefreshTokenInvalid) {
				t.Fatalf("ended family error=%v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestPushSessionUnregisterUsesOnlyOriginalOwnerOrLegacyNull(t *testing.T) {
	repo, mock, cleanup := newPushRepositoryTest(t)
	defer cleanup()
	sid := strings.Repeat("a", 32)
	mock.ExpectExec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=\? AND DEVICE_TOKEN=\? AND \(SESSION_SID=\? OR SESSION_SID IS NULL\)`).WithArgs(42, "synthetic", sid).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UnregisterDeviceForSession(42, sid, "synthetic"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
