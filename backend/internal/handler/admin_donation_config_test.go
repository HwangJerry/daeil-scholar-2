package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

const validBalanceConfigFields = `"tierSproutMin":1,"tierSaplingMin":10000,"tierTreeMin":50000,"tierBloomingMin":100000,"tierFruitingMin":300000`

func TestAdminDonationConfigRejectsInvalidBalance(t *testing.T) {
	for _, tc := range []struct{ name, fields, code string }{
		{"negative", `"balanceAmount":-1,"balanceAsOf":"2026-09-23"`, "INVALID_DONATION_BALANCE"},
		{"fraction", `"balanceAmount":1.5,"balanceAsOf":"2026-09-23"`, "INVALID_BODY"},
		{"string amount", `"balanceAmount":"1000","balanceAsOf":"2026-09-23"`, "INVALID_BODY"},
		{"overflow", `"balanceAmount":9223372036854775808,"balanceAsOf":"2026-09-23"`, "INVALID_BODY"},
		{"missing date with zero", `"balanceAmount":0`, "INVALID_DONATION_BALANCE"},
		{"null date", `"balanceAmount":1000,"balanceAsOf":null`, "INVALID_DONATION_BALANCE"},
		{"invalid date", `"balanceAmount":1000,"balanceAsOf":"2026-02-29"`, "INVALID_DONATION_BALANCE"},
		{"not ISO", `"balanceAmount":1000,"balanceAsOf":"2026-9-23"`, "INVALID_DONATION_BALANCE"},
		{"timestamp", `"balanceAmount":1000,"balanceAsOf":"2026-09-23T00:00:00Z"`, "INVALID_DONATION_BALANCE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewAdminDonationHandler(service.NewDonationConfigOrchestrator(service.NewAdminDonationService(nil, nil), nil, nil))
			r := httptest.NewRequest(http.MethodPut, "/api/admin/donation/config", strings.NewReader("{"+validBalanceConfigFields+","+tc.fields+"}"))
			r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 7}))
			w := httptest.NewRecorder()
			h.UpdateConfig(w, r)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"code":"`+tc.code+`"`) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

type balanceSnapshotCreatorStub struct{ calls int }

func (s *balanceSnapshotCreatorStub) CreateSnapshotNow() error { return nil }
func (s *balanceSnapshotCreatorStub) CreateSnapshotTx(_ *sqlx.Tx) error {
	s.calls++
	return nil
}

func TestAdminDonationConfigSavesReadsAndClearsBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	xdb := sqlx.NewDb(db, "sqlmock")
	repo := repository.NewDonationRepository(xdb)
	store := cache.New(time.Minute, time.Minute)
	donationService := service.NewDonationService(repo, store)
	snapshots := &balanceSnapshotCreatorStub{}
	h := NewAdminDonationHandler(service.NewDonationConfigOrchestrator(
		service.NewAdminDonationService(repository.NewAdminDonationRepository(xdb), repo), donationService, snapshots))
	for _, amount := range []any{int64(5000000000), int64(0), nil} {
		var date any
		if amount != nil {
			date = "2026-09-23"
		}
		fields, err := json.Marshal(map[string]any{"balanceAmount": amount, "balanceAsOf": date})
		if err != nil {
			t.Fatal(err)
		}
		store.Set("donation_summary", "previous balance", time.Minute)
		mock.ExpectBegin()
		mock.ExpectExec(`(?s)UPDATE DONATION_CONFIG.*DC_BALANCE_AMOUNT = \?, DC_BALANCE_AS_OF = \?`).
			WithArgs(0, 0, 0, 1, 10000, 50000, 100000, 300000, amount, date, "", "N", 7).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		r := httptest.NewRequest(http.MethodPut, "/api/admin/donation/config", strings.NewReader("{"+validBalanceConfigFields+","+string(fields[1:])))
		r = r.WithContext(middleware.SetAuthUser(r.Context(), &model.AuthUser{USRSeq: 7}))
		w := httptest.NewRecorder()
		h.UpdateConfig(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if _, found := store.Get("donation_summary"); found {
			t.Fatal("successful save did not invalidate public summary")
		}
		mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(sqlmock.NewRows([]string{"DC_BALANCE_AMOUNT", "DC_BALANCE_AS_OF"}).AddRow(amount, date))
		w = httptest.NewRecorder()
		h.GetConfig(w, httptest.NewRequest(http.MethodGet, "/api/admin/donation/config", nil))
		var config model.DonationConfig
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &config) != nil {
			t.Fatalf("read status=%d body=%s", w.Code, w.Body.String())
		}
		if amount == nil {
			if config.BalanceAmount != nil || config.BalanceAsOf != nil || !strings.Contains(w.Body.String(), `"dcBalanceAmount":null`) {
				t.Fatalf("cleared config = %s", w.Body.String())
			}
		} else if config.BalanceAmount == nil || *config.BalanceAmount != amount || config.BalanceAsOf == nil || *config.BalanceAsOf != date {
			t.Fatalf("saved config = %s", w.Body.String())
		}
	}
	if snapshots.calls != 3 {
		t.Fatalf("snapshot refresh calls=%d, want 3", snapshots.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminDonationUpdateConfigReturnsBadRequestForInvalidTierThresholds(t *testing.T) {
	adminService := service.NewAdminDonationService(nil, nil)
	orchestrator := service.NewDonationConfigOrchestrator(adminService, nil, nil)
	handler := NewAdminDonationHandler(orchestrator)
	request := httptest.NewRequest(http.MethodPut, "/api/admin/donation/config", strings.NewReader(`{
		"goal":200000000,
		"manualAdj":0,
		"manualDonorCnt":0,
		"tierSproutMin":1,
		"tierSaplingMin":10000,
		"tierTreeMin":50000,
		"tierBloomingMin":50000,
		"tierFruitingMin":300000,
		"note":"invalid thresholds",
		"overwrite":false
	}`))
	request = request.WithContext(middleware.SetAuthUser(request.Context(), &model.AuthUser{USRSeq: 7}))
	response := httptest.NewRecorder()

	handler.UpdateConfig(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INVALID_TIER_THRESHOLDS"`) {
		t.Fatalf("body = %s, want INVALID_TIER_THRESHOLDS", response.Body.String())
	}
}
