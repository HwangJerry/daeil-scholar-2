// mobile_session_principal_test.go — Verifies sessions embed current canonical authorization state.
package service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
)

func TestMobileSessionIssuerLoadsPendingCanonicalPrincipal(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()

	mock.ExpectQuery(`LEFT JOIN ALUMNI_VERIFICATION`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "EMAIL", "ADMIN_ROLE",
			"VERIFICATION_STATUS", "GRADUATION_YEAR", "COHORT", "DEPARTMENT",
			"REJECTION_REASON", "SUBMITTED_AT", "REVIEWED_AT",
		}).AddRow(
			42, "legacy-id", "Pending Member", "BBB", "pending@example.com", nil,
			"pending", nil, "18", "영어", nil, nil, nil,
		))
	mock.ExpectExec(`INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN`).
		WithArgs(sqlmock.AnyArg(), 42, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	session, err := NewMobileSessionIssuer(auth).Issue(&model.User{
		USRSeq:    42,
		USRID:     "legacy-id",
		USRName:   "Pending Member",
		USRStatus: "BBB",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.User.Email != "pending@example.com" || session.User.Verification.Status != model.VerificationPending {
		t.Fatalf("session principal = %#v", session.User)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("pending member must receive the normal token shape")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
