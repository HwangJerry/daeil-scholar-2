package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

func TestWrongPasswordCanRetryWithSameSocialLinkToken(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	memberService := NewMemberService(auth.repo)
	store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := store.Put("token", model.SocialLinkData{SocialID: "subject"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	params := mergeLinkParams()

	mock.ExpectQuery(`WHERE USR_ID = \? AND USR_PWD = \?`).
		WithArgs("member", MysqlNativePassword("wrong-password")).
		WillReturnError(sql.ErrNoRows)
	firstLease, err := store.Begin("token")
	if err != nil {
		t.Fatal(err)
	}
	params.ExistingPassword = "wrong-password"
	if _, _, err := auth.LinkSocialAccount(params, memberService); !errors.Is(err, ErrExistingAccountReauthenticationRequired) {
		t.Fatalf("wrong password error = %v", err)
	}
	if err := store.Release(firstLease); err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`WHERE USR_ID = \? AND USR_PWD = \?`).
		WithArgs("member", MysqlNativePassword("correct-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", "010-1234-5678", "31", "member@example.com", nil, nil))
	expectMergeSocialAccountTransaction(mock)

	retryLease, err := store.Begin("token")
	if err != nil {
		t.Fatalf("same token was not retryable: %v", err)
	}
	params.ExistingPassword = "correct-password"
	user, _, err := auth.LinkSocialAccount(params, memberService)
	if err != nil {
		t.Fatal(err)
	}
	if user.USRSeq != 42 {
		t.Fatalf("user = %#v", user)
	}
	if err := store.Consume(retryLease); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin("token"); !errors.Is(err, ErrSocialLinkTokenConsumed) {
		t.Fatalf("successful token replay error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSocialLinkTransactionFailureLeavesTokenRetryable(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	memberService := NewMemberService(auth.repo)
	store := NewSocialLinkTokenStore(cache.New(time.Minute, time.Minute))
	if _, err := store.Put("token", model.SocialLinkData{}, time.Minute); err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`WHERE \(USR_PHONE = \? OR`).
		WithArgs("01012345678", "01012345678").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO WEO_MEMBER`).
		WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()

	lease, err := store.Begin("token")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = auth.LinkSocialAccount(newLinkParams(), memberService)
	if err == nil {
		t.Fatal("database failure must escape")
	}
	if err := store.Release(lease); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin("token"); err != nil {
		t.Fatalf("database rollback must preserve retryability: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMergeSocialAccountRejectsProviderOwnedByAnotherMember(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	memberService := NewMemberService(auth.repo)
	params := mergeLinkParams()
	params.ExistingPassword = "correct-password"

	mock.ExpectQuery(`WHERE USR_ID = \? AND USR_PWD = \?`).
		WithArgs("member", MysqlNativePassword("correct-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", "010-1234-5678", "31", "member@example.com", nil, nil))
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "subject").
		WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ"}).AddRow(99))
	mock.ExpectRollback()

	_, _, err := auth.LinkSocialAccount(params, memberService)
	if err == nil || err.Error() != "account merge not supported" {
		t.Fatalf("merge error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMergeSocialAccountReauthenticatesCanonicalEmailAndPreservesProfile(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	memberService := NewMemberService(auth.repo)
	params := SocialLinkParams{
		Mode:                SocialLinkModeMerge,
		Provider:            "KT",
		SocialID:            "subject",
		Email:               "provider@example.com",
		ExistingEmail:       "member@example.com",
		ExistingPassword:    "correct-password",
		EncryptedCredential: "encrypted",
	}

	mock.ExpectQuery(`LOWER\(TRIM\(USR_EMAIL\)\) = LOWER\(TRIM\(\?\)\) AND USR_PWD = \?`).
		WithArgs("member@example.com", MysqlNativePassword("correct-password")).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(
			42, "legacy-id", "Member", "CCC", "010-1234-5678",
			"31", "member@example.com", nil, nil,
		))
	expectAttachSocialAccountTransaction(mock)

	user, isNew, err := auth.LinkSocialAccount(params, memberService)
	if err != nil {
		t.Fatal(err)
	}
	if isNew || user.USRSeq != 42 {
		t.Fatalf("user = %#v, isNew = %v", user, isNew)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func mergeLinkParams() SocialLinkParams {
	params := newLinkParams()
	params.Mode = SocialLinkModeMerge
	params.ExistingUSRID = "member"
	params.EncryptedCredential = "encrypted"
	return params
}

func newLinkParams() SocialLinkParams {
	return SocialLinkParams{
		Mode:            SocialLinkModeNew,
		Provider:        "KT",
		SocialID:        "subject",
		Email:           "member@example.com",
		Name:            "Member",
		Phone:           "010-1234-5678",
		FN:              "31",
		FmDept:          "영어",
		USRPhonePublic:  "N",
		USREmailPublic:  "N",
		ProfileImageURL: "",
	}
}

func expectMergeSocialAccountTransaction(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE WEO_MEMBER`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_CREDENTIAL`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`FROM WEO_MEMBER`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "member", "Member", "CCC", "01012345678", "31", "member@example.com", nil, nil))
	mock.ExpectCommit()
}

func expectAttachSocialAccountTransaction(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM WEO_MEMBER_SOCIAL`).
		WithArgs("KT", "subject").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO WEO_MEMBER_SOCIAL`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO ALUMNI_SOCIAL_CREDENTIAL`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`FROM WEO_MEMBER`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{
			"USR_SEQ", "USR_ID", "USR_NAME", "USR_STATUS", "USR_PHONE",
			"USR_FN", "USR_EMAIL", "USR_NICK", "USR_PHOTO",
		}).AddRow(42, "legacy-id", "Member", "CCC", "01012345678", "31", "member@example.com", nil, nil))
	mock.ExpectCommit()
}
