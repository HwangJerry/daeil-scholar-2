package service

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func TestRejectAlumniVerificationRequiresReason(t *testing.T) {
	service := NewAdminMemberService(nil)

	err := service.RejectAlumniVerification(42, 7, " \n\t ", time.Now())
	if !errors.Is(err, ErrRejectionReasonRequired) {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectAlumniVerificationRecordsCanonicalReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock")))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)

	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("rejected", "확인 가능한 학적 정보를 입력해주세요.", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.RejectAlumniVerification(
		42,
		7,
		"  확인 가능한 학적 정보를 입력해주세요.  ",
		expectedUpdatedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectAlumniVerificationDetectsStaleReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock")))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)
	currentUpdatedAt := expectedUpdatedAt.Add(time.Minute)

	mock.ExpectExec(`UPDATE ALUMNI_VERIFICATION`).
		WithArgs("rejected", "사유", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT STATUS, UPDATED_AT`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"STATUS", "UPDATED_AT"}).AddRow("pending", currentUpdatedAt))

	err = service.RejectAlumniVerification(42, 7, "사유", expectedUpdatedAt)
	if !errors.Is(err, ErrVerificationStale) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApproveAlumniVerificationCapturesApprovedAcademicSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewAdminMemberService(repository.NewAdminMemberRepository(sqlx.NewDb(db, "sqlmock")))
	expectedUpdatedAt := time.Date(2026, time.July, 27, 1, 0, 0, 0, time.UTC)

	mock.ExpectExec(`APPROVED_GRADUATION_YEAR = GRADUATION_YEAR`).
		WithArgs("approved", 7, 42, expectedUpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.ApproveAlumniVerification(42, 7, expectedUpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMemberStatusCannotBypassCanonicalVerification(t *testing.T) {
	service := NewAdminMemberService(nil)
	for _, status := range []string{"BAA", "BBB", "CCC", "ZZZ"} {
		err := service.UpdateStatus(42, status)
		if !errors.Is(err, ErrLegacyVerificationStatusNotAllowed) {
			t.Fatalf("status %s error = %v", status, err)
		}
	}
}
