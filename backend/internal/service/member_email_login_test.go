// member_email_login_test.go — Verifies canonical email/password service login.
package service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoginWithEmailPasswordReturnsPendingLifecycleMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	password := "correct-password"

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("member@example.com", MysqlNativePassword(password)).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "legacy-id", "Pending Member", "BBB", nil, nil, "member@example.com", nil, nil))

	user, err := NewMemberService(auth.repo).LoginWithEmailPassword("member@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	if user == nil || user.USRSeq != 42 || user.USRStatus != "BBB" {
		t.Fatalf("user = %#v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
