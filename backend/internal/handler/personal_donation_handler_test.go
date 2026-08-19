package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type personalDonationHandlerRepositoryStub struct {
	usrSeq int
}

func (s *personalDonationHandlerRepositoryStub) GetTotals(usrSeq int) (int64, int, error) {
	s.usrSeq = usrSeq
	return 80000, 1, nil
}

func (s *personalDonationHandlerRepositoryStub) List(usrSeq int, _ string, _, _ int) ([]model.PersonalDonationItem, error) {
	s.usrSeq = usrSeq
	return []model.PersonalDonationItem{{OrderSeq: 3001, NetReceivedAmount: 80000}}, nil
}

func TestPersonalDonationHandlerUsesAuthenticatedUserSequence(t *testing.T) {
	repo := &personalDonationHandlerRepositoryStub{}
	personalDonationHandler := NewPersonalDonationHandler(service.NewPersonalDonationService(repo))
	request := httptest.NewRequest(http.MethodGet, "/api/donation/my?page=1&size=20", nil)
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 42}))
	recorder := httptest.NewRecorder()

	personalDonationHandler.GetMyDonations(recorder, request)

	if recorder.Code != http.StatusOK || repo.usrSeq != 42 {
		t.Fatalf("status/user = %d/%d, body = %s", recorder.Code, repo.usrSeq, recorder.Body.String())
	}
	for _, expected := range []string{`"totalAmount":80000`, `"netReceivedAmount":80000`, `"page":1`, `"size":20`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("body = %s, missing %s", recorder.Body.String(), expected)
		}
	}
}

func TestPersonalDonationHandlerRequiresAuthentication(t *testing.T) {
	personalDonationHandler := NewPersonalDonationHandler(service.NewPersonalDonationService(&personalDonationHandlerRepositoryStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/donation/my", nil)
	recorder := httptest.NewRecorder()

	personalDonationHandler.GetMyDonations(recorder, request)

	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
