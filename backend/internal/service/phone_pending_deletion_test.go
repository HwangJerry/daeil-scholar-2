package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

// pendingDeletionMemberRepo marks one number as belonging to an account awaiting erasure.
type pendingDeletionMemberRepo struct {
	stubMemberRepository
	pendingPhone string
	existing     *model.User
}

func (r *pendingDeletionMemberRepo) PhoneHasPendingDeletion(phone string) (bool, error) {
	return phone == r.pendingPhone, nil
}

func (r *pendingDeletionMemberRepo) FindMemberByPhone(string) (*model.User, error) {
	return r.existing, nil
}

func TestSocialSignupRejectsNumberPendingDeletion(t *testing.T) {
	memberSvc := &MemberService{repo: &pendingDeletionMemberRepo{pendingPhone: "01012345678"}}
	_, created, err := (&AuthService{}).LinkSocialAccount(SocialLinkParams{Phone: "010-1234-5678"}, memberSvc)
	if !errors.Is(err, ErrPhonePendingDeletion) || created {
		t.Fatalf("expected ErrPhonePendingDeletion without creating a member, got created=%v err=%v", created, err)
	}
}

func TestSocialSignupStillReportsActiveOwnerForOtherNumbers(t *testing.T) {
	memberSvc := &MemberService{repo: &pendingDeletionMemberRepo{pendingPhone: "01099999999", existing: &model.User{USRSeq: 7}}}
	_, _, err := (&AuthService{}).LinkSocialAccount(SocialLinkParams{Phone: "01012345678"}, memberSvc)
	if !errors.Is(err, ErrOwnershipConfirmationRequired) {
		t.Fatalf("a number owned by an active member keeps the existing error, got %v", err)
	}
}

type pendingDeletionRegistrationRepo struct {
	stubRegistrationMemberRepository
	phoneExists     bool
	pendingDeletion bool
}

func (r *pendingDeletionRegistrationRepo) CheckPhoneExists(string) (bool, error) {
	return r.phoneExists, nil
}
func (r *pendingDeletionRegistrationRepo) PhoneHasPendingDeletion(string) (bool, error) {
	return r.pendingDeletion, nil
}

func TestRegisterDistinguishesPendingDeletionFromTakenNumber(t *testing.T) {
	cases := []struct {
		name    string
		pending bool
		want    error
	}{
		{"pending deletion", true, ErrPhonePendingDeletion},
		{"active owner", false, ErrPhoneTaken},
	}
	for _, tc := range cases {
		svc := &RegistrationService{memberRepo: &pendingDeletionRegistrationRepo{phoneExists: true, pendingDeletion: tc.pending}}
		_, err := svc.Register(model.RegisterRequest{UsrID: "member1", Phone: "01012345678"})
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, err, tc.want)
		}
	}
}
