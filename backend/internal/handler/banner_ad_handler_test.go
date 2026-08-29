package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

const bannerEventExpectationTimeout = time.Second

func TestBannerAdGetActiveReturnsActiveBanner(t *testing.T) {
	handler, mock := newBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).
		WillReturnRows(sqlmock.NewRows([]string{
			"BN_SEQ", "BN_NAME", "BN_URL", "BN_START_DATE", "BN_END_DATE",
		}).AddRow(7, "여름 배너", "https://example.com/banner", nil, nil))
	mock.ExpectQuery(`FROM MAIN_BANNER_AD_IMAGE`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{
			"BNI_SEQ", "BN_SEQ", "IMAGE_URL", "SORT_ORDER",
		}).AddRow(21, 7, "/uploads/bannerAd/summer.png", 0))

	request := httptest.NewRequest(http.MethodGet, "/api/banner-ads/active", nil)
	response := httptest.NewRecorder()

	handler.GetActive(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, want := range []string{
		`"bnSeq":7`,
		`"bnName":"여름 배너"`,
		`"imageUrl":"/uploads/bannerAd/summer.png"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("body = %s, want substring %s", response.Body.String(), want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBannerAdGetActiveReturnsEmptyBodyWhenNoBannerExists(t *testing.T) {
	handler, mock := newBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).WillReturnError(sql.ErrNoRows)

	request := httptest.NewRequest(http.MethodGet, "/api/banner-ads/active", nil)
	response := httptest.NewRecorder()

	handler.GetActive(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty body", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBannerAdGetActiveMapsServiceErrorToInternalServerError(t *testing.T) {
	handler, mock := newBannerAdHandlerForTest(t)
	mock.ExpectQuery(`FROM MAIN_BANNER_AD`).WillReturnError(errors.New("database unavailable"))

	request := httptest.NewRequest(http.MethodGet, "/api/banner-ads/active", nil)
	response := httptest.NewRecorder()

	handler.GetActive(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"FETCH_FAILED"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBannerAdTrackEventsReturnNoContent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		track     func(*BannerAdHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:      "view",
			eventType: "VIEW",
			track: func(handler *BannerAdHandler, response http.ResponseWriter, request *http.Request) {
				handler.TrackView(response, request)
			},
		},
		{
			name:      "click",
			eventType: "CLICK",
			track: func(handler *BannerAdHandler, response http.ResponseWriter, request *http.Request) {
				handler.TrackClick(response, request)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, mock := newBannerAdHandlerForTest(t)
			mock.ExpectExec(`INSERT INTO WEO_BANNER_AD_LOG`).
				WithArgs(7, test.eventType).
				WillReturnResult(sqlmock.NewResult(1, 1))

			request := httptest.NewRequest(http.MethodPost, "/api/banner-ads/7/"+test.name, nil)
			request = withBannerAdRouteParam(request, "bnSeq", "7")
			response := httptest.NewRecorder()

			test.track(handler, response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			waitForBannerAdExpectations(t, mock)
		})
	}
}

func TestBannerAdTrackEventsRejectInvalidBNSeq(t *testing.T) {
	tests := []struct {
		name  string
		track func(*BannerAdHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name: "view",
			track: func(handler *BannerAdHandler, response http.ResponseWriter, request *http.Request) {
				handler.TrackView(response, request)
			},
		},
		{
			name: "click",
			track: func(handler *BannerAdHandler, response http.ResponseWriter, request *http.Request) {
				handler.TrackClick(response, request)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewBannerAdHandler(service.NewBannerAdService(nil), zerolog.Nop())
			request := httptest.NewRequest(http.MethodPost, "/api/banner-ads/not-a-number/"+test.name, nil)
			request = withBannerAdRouteParam(request, "bnSeq", "not-a-number")
			response := httptest.NewRecorder()

			test.track(handler, response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"code":"INVALID_SEQ"`) {
				t.Fatalf("body = %s", response.Body.String())
			}
		})
	}
}

func newBannerAdHandlerForTest(t *testing.T) (*BannerAdHandler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := repository.NewBannerAdRepository(sqlx.NewDb(db, "sqlmock"))
	return NewBannerAdHandler(service.NewBannerAdService(repo), zerolog.Nop()), mock
}

func withBannerAdRouteParam(request *http.Request, name, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func waitForBannerAdExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	deadline := time.Now().Add(bannerEventExpectationTimeout)
	for {
		if err := mock.ExpectationsWereMet(); err == nil {
			return
		} else if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
}
