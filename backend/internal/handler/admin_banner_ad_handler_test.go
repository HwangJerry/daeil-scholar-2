package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
)

func TestAdminBannerAdListReturnsBanners(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).
		WillReturnRows(sqlmock.NewRows([]string{
			"BN_SEQ", "BN_NAME", "BN_URL", "OPEN_YN", "INDX", "BN_START_DATE", "BN_END_DATE", "CREATED_AT", "UPDATED_AT",
		}).AddRow(7, "여름 배너", "https://example.com/banner", "Y", 1, nil, nil, "2026-08-29 10:00:00", "2026-08-29 10:00:00"))
	mock.ExpectQuery(`FROM MAIN_BANNER_AD_IMAGE`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BNI_SEQ", "BN_SEQ", "IMAGE_URL", "SORT_ORDER",
		}).AddRow(21, 7, "/uploads/bannerAd/summer.png", 0))
	mock.ExpectQuery(`FROM WEO_BANNER_AD_LOG`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BN_SEQ", "VIEW_COUNT", "CLICK_COUNT",
		}).AddRow(7, 13, 5))

	request := httptest.NewRequest(http.MethodGet, "/api/admin/banner-ads", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, want := range []string{
		`"bnSeq":7`,
		`"bnName":"여름 배너"`,
		`"viewCount":13`,
		`"clickCount":5`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("body = %s, want substring %s", response.Body.String(), want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBannerAdDetailReturnsBanner(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BN_SEQ", "BN_NAME", "BN_URL", "OPEN_YN", "INDX", "BN_START_DATE", "BN_END_DATE", "CREATED_AT", "UPDATED_AT",
		}).AddRow(7, "여름 배너", "https://example.com/banner", "Y", 1, nil, nil, "2026-08-29 10:00:00", "2026-08-29 10:00:00"))
	mock.ExpectQuery(`FROM MAIN_BANNER_AD_IMAGE`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BNI_SEQ", "BN_SEQ", "IMAGE_URL", "SORT_ORDER",
		}).AddRow(21, 7, "/uploads/bannerAd/summer.png", 0))
	mock.ExpectQuery(`FROM WEO_BANNER_AD_LOG`).
		WithArgs(7, 7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BN_SEQ", "VIEW_COUNT", "CLICK_COUNT",
		}).AddRow(7, 13, 5))

	request := httptest.NewRequest(http.MethodGet, "/api/admin/banner-ads/7", nil)
	request = withBannerAdRouteParam(request, "bnSeq", "7")
	response := httptest.NewRecorder()

	handler.Detail(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, want := range []string{`"bnSeq":7`, `"imageUrl":"/uploads/bannerAd/summer.png"`, `"viewCount":13`} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("body = %s, want substring %s", response.Body.String(), want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBannerAdDetailReturnsNotFound(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).WithArgs(404).WillReturnError(sql.ErrNoRows)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/banner-ads/404", nil)
	request = withBannerAdRouteParam(request, "bnSeq", "404")
	response := httptest.NewRecorder()

	handler.Detail(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"NOT_FOUND"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBannerAdDetailRejectsInvalidBNSeq(t *testing.T) {
	handler := NewAdminBannerAdHandler(service.NewAdminBannerAdService(nil))
	request := httptest.NewRequest(http.MethodGet, "/api/admin/banner-ads/not-a-number", nil)
	request = withBannerAdRouteParam(request, "bnSeq", "not-a-number")
	response := httptest.NewRecorder()

	handler.Detail(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_SEQ"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminBannerAdCreateMapsInvalidURLToBadRequest(t *testing.T) {
	handler := NewAdminBannerAdHandler(service.NewAdminBannerAdService(nil))
	request := httptest.NewRequest(http.MethodPost, "/api/admin/banner-ads", strings.NewReader(`{
		"bnName":"잘못된 배너",
		"bnUrl":"ftp://example.com/banner",
		"openYn":"N",
		"indx":1,
		"imageUrls":[]
	}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_URL"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminBannerAdCreateMapsActiveConflictToConflict(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(0, nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectRollback()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/banner-ads", strings.NewReader(validActiveBannerRequest()))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"ACTIVE_CONFLICT"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBannerAdUpdateMapsInvalidURLToBadRequest(t *testing.T) {
	handler := NewAdminBannerAdHandler(service.NewAdminBannerAdService(nil))
	request := httptest.NewRequest(http.MethodPut, "/api/admin/banner-ads/7", strings.NewReader(`{
		"bnName":"잘못된 배너",
		"bnUrl":"ftp://example.com/banner",
		"openYn":"N",
		"indx":1,
		"imageUrls":[]
	}`))
	request = withBannerAdRouteParam(request, "seq", "7")
	response := httptest.NewRecorder()

	handler.Update(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_URL"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestAdminBannerAdUpdateMapsActiveConflictToConflict(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IMAGE_URL`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"IMAGE_URL"}))
	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(7, nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectRollback()
	request := httptest.NewRequest(http.MethodPut, "/api/admin/banner-ads/7", strings.NewReader(validActiveBannerRequest()))
	request = withBannerAdRouteParam(request, "seq", "7")
	response := httptest.NewRecorder()

	handler.Update(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"ACTIVE_CONFLICT"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBannerAdDeleteReturnsNoContent(t *testing.T) {
	handler, mock := newAdminBannerAdHandlerForTest(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT IMAGE_URL`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"IMAGE_URL"}))
	mock.ExpectExec(`DELETE FROM MAIN_BANNER_AD`).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	request := httptest.NewRequest(http.MethodDelete, "/api/admin/banner-ads/7", nil)
	request = withBannerAdRouteParam(request, "seq", "7")
	response := httptest.NewRecorder()

	handler.Delete(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty body", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newAdminBannerAdHandlerForTest(t *testing.T) (*AdminBannerAdHandler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := repository.NewAdminBannerAdRepository(sqlx.NewDb(db, "sqlmock"))
	return NewAdminBannerAdHandler(service.NewAdminBannerAdService(repo)), mock
}

func validActiveBannerRequest() string {
	return `{
		"bnName":"활성 배너",
		"bnUrl":"https://example.com/banner",
		"openYn":"Y",
		"indx":1,
		"imageUrls":["/uploads/bannerAd/banner.png"]
	}`
}
