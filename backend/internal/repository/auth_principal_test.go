// auth_principal_test.go — Verifies canonical principal loading from member, verification, and role tables.
package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func TestGetAuthPrincipalBySeqLoadsAuthorizationState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
	submittedAt := time.Date(2026, 7, 27, 1, 0, 0, 0, time.UTC)
	reviewedAt := submittedAt.Add(time.Hour)

	mock.ExpectQuery(`LEFT JOIN ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "EMAIL", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(
			42, "legacy-id", "Member", "CCC", "member@example.com", "operator",
			"approved", 2003, "18", "영어", nil, submittedAt, reviewedAt,
		))

	principal, err := repo.GetAuthPrincipalBySeq(42)
	if err != nil {
		t.Fatal(err)
	}
	if principal == nil {
		t.Fatal("principal is nil")
	}
	if principal.Email != "member@example.com" || principal.AdminRole == nil || *principal.AdminRole != model.AdminRoleOperator {
		t.Fatalf("principal identity/role = %#v", principal)
	}
	if principal.Verification.Status != model.VerificationApproved || principal.Verification.GraduationYear == nil || *principal.Verification.GraduationYear != 2003 {
		t.Fatalf("principal verification = %#v", principal.Verification)
	}
	if principal.Verification.SubmittedAt == nil || !principal.Verification.SubmittedAt.Equal(submittedAt) || principal.Verification.ReviewedAt == nil || !principal.Verification.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf("principal timestamps = %#v", principal.Verification)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
