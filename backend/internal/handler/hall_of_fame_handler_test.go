// hall_of_fame_handler_test.go — Public cache headers and the no-counts response contract.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

type stubHallOfFameRepo struct{ entries []model.HallOfFameEntry }

func (s stubHallOfFameRepo) ListPublic() ([]model.HallOfFameEntry, error) { return s.entries, nil }
func (s stubHallOfFameRepo) ListAdmin(model.HallOfFameAdminFilter) ([]model.HallOfFameEntry, int, []model.HallOfFameTierCount, error) {
	return nil, 0, nil, nil
}
func (s stubHallOfFameRepo) FindActiveByID(int64) (*model.HallOfFameEntry, error) {
	return nil, nil
}
func (s stubHallOfFameRepo) FindByNaturalKey(string) (*model.HallOfFameEntry, error) {
	return nil, nil
}
func (s stubHallOfFameRepo) Insert(model.HallOfFameEntry) (int64, error) { return 0, nil }
func (s stubHallOfFameRepo) Save(model.HallOfFameEntry) error            { return nil }
func (s stubHallOfFameRepo) SoftDeleteManual(int64) error                { return nil }

func TestHallOfFameGetCachingAndShape(t *testing.T) {
	tier, cohort, department := 1000, 1, "독일어과"
	h := NewHallOfFameHandler(service.NewHallOfFameService(stubHallOfFameRepo{entries: []model.HallOfFameEntry{
		{ID: 1, DisplayName: "김하나", Kind: model.HallOfFameKindAlumni, Cohort: &cohort, Department: &department, TierManWon: &tier},
	}}))

	recorder := httptest.NewRecorder()
	h.Get(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/hall-of-fame", nil))
	if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("status %d headers %v", recorder.Code, recorder.Header())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `{"tierManWon":1000,"entries":[{"displayName":"김하나","label":"1기 · 독일어과"}]}`) {
		t.Fatalf("body: %s", body)
	}
	etag := recorder.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/hall-of-fame", nil)
	request.Header.Set("If-None-Match", etag)
	recorder = httptest.NewRecorder()
	h.Get(recorder, request)
	if recorder.Code != http.StatusNotModified || recorder.Body.Len() != 0 {
		t.Fatalf("conditional: %d", recorder.Code)
	}
}
