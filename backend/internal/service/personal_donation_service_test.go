package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type personalDonationRepositoryStub struct {
	totalAmount int64
	totalCount  int
	items       []model.PersonalDonationItem
	usrSeq      int
	sort        string
	page        int
	size        int
}

func (s *personalDonationRepositoryStub) GetTotals(usrSeq int) (int64, int, error) {
	s.usrSeq = usrSeq
	return s.totalAmount, s.totalCount, nil
}

func (s *personalDonationRepositoryStub) List(usrSeq int, sort string, page, size int) ([]model.PersonalDonationItem, error) {
	s.usrSeq = usrSeq
	s.sort = sort
	s.page = page
	s.size = size
	return s.items, nil
}

func TestPersonalDonationServiceReturnsCanonicalSummary(t *testing.T) {
	repo := &personalDonationRepositoryStub{
		totalAmount: 80000,
		totalCount:  1,
		items: []model.PersonalDonationItem{{
			OrderSeq: 3001, GrossAmount: 100000, RefundedAmount: 20000,
			NetReceivedAmount: 80000, LifecycleStatus: "partially_refunded",
		}},
	}
	personalDonationService := NewPersonalDonationService(repo)

	result, err := personalDonationService.GetPersonalDonations(42, "latest", 2, 10)
	if err != nil {
		t.Fatalf("GetPersonalDonations() error = %v", err)
	}
	if result.TotalAmount != 80000 || result.TotalCount != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repo.usrSeq != 42 || repo.sort != "latest" || repo.page != 2 || repo.size != 10 {
		t.Fatalf("repository args = %d/%s/%d/%d", repo.usrSeq, repo.sort, repo.page, repo.size)
	}
}

func TestPersonalDonationServiceReturnsEmptyListForUserWithoutDonations(t *testing.T) {
	repo := &personalDonationRepositoryStub{items: make([]model.PersonalDonationItem, 0)}
	personalDonationService := NewPersonalDonationService(repo)

	result, err := personalDonationService.GetPersonalDonations(42, "", 0, 0)
	if err != nil {
		t.Fatalf("GetPersonalDonations() error = %v", err)
	}
	if result.Items == nil || len(result.Items) != 0 || result.TotalAmount != 0 || result.TotalCount != 0 {
		t.Fatalf("unexpected empty result: %+v", result)
	}
	if result.Page != 1 || result.Size != 20 || repo.sort != "latest" {
		t.Fatalf("defaults = page %d size %d sort %s", result.Page, result.Size, repo.sort)
	}
}

func TestPersonalDonationServiceNormalizesInvalidPagination(t *testing.T) {
	repo := &personalDonationRepositoryStub{items: make([]model.PersonalDonationItem, 0)}
	personalDonationService := NewPersonalDonationService(repo)

	result, err := personalDonationService.GetPersonalDonations(42, "amount", -3, 51)
	if err != nil {
		t.Fatalf("GetPersonalDonations() error = %v", err)
	}
	if result.Page != 1 || result.Size != 20 || repo.sort != "amount" || repo.page != 1 || repo.size != 20 {
		t.Fatalf("normalized args/result = %+v, repo %s/%d/%d", result, repo.sort, repo.page, repo.size)
	}
}

func TestPersonalDonationServiceRejectsInvalidUser(t *testing.T) {
	personalDonationService := NewPersonalDonationService(&personalDonationRepositoryStub{})
	_, err := personalDonationService.GetPersonalDonations(0, "", 1, 20)
	if !errors.Is(err, ErrInvalidPersonalDonationUser) {
		t.Fatalf("error = %v, want ErrInvalidPersonalDonationUser", err)
	}
}
