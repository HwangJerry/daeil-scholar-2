package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type memberBlockRepoStub struct {
	listItems      []model.MemberBlockState
	listCalled     bool
	getCalled      bool
	blockCalled    bool
	blockErr       error
	unblockCalled  bool
	approved       bool
	approvalCalled bool
}

func (s *memberBlockRepoStub) IsApprovedAlumni(int) (bool, error) {
	s.approvalCalled = true
	return s.approved, nil
}

func (s *memberBlockRepoStub) List(int) ([]model.MemberBlockState, error) {
	s.listCalled = true
	return s.listItems, nil
}
func (s *memberBlockRepoStub) Get(int, int) (*model.MemberBlockState, error) {
	s.getCalled = true
	return nil, nil
}
func (s *memberBlockRepoStub) Block(int, int) (*model.MemberBlockState, error) {
	s.blockCalled = true
	return nil, s.blockErr
}
func (s *memberBlockRepoStub) Unblock(int, int) (*model.MemberBlockState, error) {
	s.unblockCalled = true
	return nil, nil
}

func TestMemberBlockServiceListNormalizesNilItems(t *testing.T) {
	repo := &memberBlockRepoStub{}
	result, err := (&MemberBlockService{repo: repo}).List(101)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty slice", result.Items)
	}
}

func TestMemberBlockServiceRejectsSelfBlockBeforeRepository(t *testing.T) {
	repo := &memberBlockRepoStub{}
	_, err := (&MemberBlockService{repo: repo}).Block(101, 101)
	if err == nil {
		t.Fatal("Block error = nil, want validation error")
	}
	if repo.blockCalled {
		t.Fatal("repository called for self block")
	}
}

func TestMemberBlockServiceRejectsUnapprovedTargetBeforeBlockInsert(t *testing.T) {
	repo := &memberBlockRepoStub{approved: false}
	_, err := (&MemberBlockService{repo: repo}).Block(101, 202)
	if err == nil {
		t.Fatal("Block error = nil, want unapproved target error")
	}
	if !repo.approvalCalled {
		t.Fatal("approval authority was not queried")
	}
	if repo.blockCalled {
		t.Fatal("block row inserted for unapproved target")
	}
}

func TestMemberBlockServiceMapsApprovalChangeDuringInsertToCanonicalNotFound(t *testing.T) {
	repo := &memberBlockRepoStub{approved: true, blockErr: repository.ErrMemberBlockTargetNotApproved}
	_, err := (&MemberBlockService{repo: repo}).Block(101, 202)
	if !errors.Is(err, ErrMemberBlockTargetNotFound) {
		t.Fatalf("error = %v, want canonical target-not-found", err)
	}
}

func TestMemberBlockServicePersistsApprovedTarget(t *testing.T) {
	repo := &memberBlockRepoStub{approved: true}
	if _, err := (&MemberBlockService{repo: repo}).Block(101, 202); err != nil {
		t.Fatal(err)
	}
	if !repo.approvalCalled || !repo.blockCalled {
		t.Fatalf("approvalCalled = %v, blockCalled = %v", repo.approvalCalled, repo.blockCalled)
	}
}

func TestMemberBlockServiceGetDoesNotQueryTargetApproval(t *testing.T) {
	repo := &memberBlockRepoStub{}
	if _, err := (&MemberBlockService{repo: repo}).Get(101, 202); err != nil {
		t.Fatal(err)
	}
	if repo.approvalCalled || !repo.getCalled {
		t.Fatalf("approvalCalled = %v, getCalled = %v", repo.approvalCalled, repo.getCalled)
	}
}

func TestMemberBlockServiceUnblockDoesNotQueryTargetApproval(t *testing.T) {
	repo := &memberBlockRepoStub{}
	if _, err := (&MemberBlockService{repo: repo}).Unblock(101, 202); err != nil {
		t.Fatal(err)
	}
	if repo.approvalCalled || !repo.unblockCalled {
		t.Fatalf("approvalCalled = %v, unblockCalled = %v", repo.approvalCalled, repo.unblockCalled)
	}
}
