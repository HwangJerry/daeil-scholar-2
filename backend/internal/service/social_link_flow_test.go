package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/patrickmn/go-cache"
)

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
