package service

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

func TestDonationSummaryMonthRolloverUsesSeoulAndBypassesPreviousMonthCache(t *testing.T) {
	for _, tc := range []struct {
		name, beforeUTC, firstStart, firstEnd, secondEnd string
	}{
		{"month", "2026-09-30T14:59:59Z", "2026-09-01", "2026-10-01", "2026-11-01"},
		{"year", "2026-12-31T14:59:59Z", "2026-12-01", "2027-01-01", "2027-02-01"},
		{"leap February", "2024-02-29T14:59:59Z", "2024-02-01", "2024-03-01", "2024-04-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			svc := NewDonationService(repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock")), cache.New(time.Minute, time.Minute))
			before, err := time.Parse(time.RFC3339, tc.beforeUTC)
			if err != nil {
				t.Fatal(err)
			}
			for index, period := range [][2]string{{tc.firstStart, tc.firstEnd}, {tc.firstEnd, tc.secondEnd}} {
				now := before.Add(time.Duration(index) * time.Second)
				mock.ExpectQuery(`FROM DONATION_SNAPSHOT`).WillReturnRows(sqlmock.NewRows([]string{"DS_TOTAL", "DS_DATE"}).AddRow(100000, tc.firstStart))
				mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(sqlmock.NewRows([]string{"DC_BALANCE_AMOUNT", "DC_BALANCE_AS_OF"}).AddRow(0, "2026-09-23"))
				amount := int64(30000)
				if index == 1 {
					amount = 0
				}
				mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_DONATION_DATE >= \? AND O_DONATION_DATE < \?`).
					WithArgs(period[0], period[1]).WillReturnRows(sqlmock.NewRows([]string{"AMOUNT"}).AddRow(amount))
				summary, err := svc.getSummaryAt(now)
				if err != nil {
					t.Fatal(err)
				}
				if summary.MonthAmount != amount || summary.BalanceAmount == nil || *summary.BalanceAmount != 0 || summary.BalanceAsOf == nil || *summary.BalanceAsOf != "2026-09-23" {
					t.Fatalf("summary = %+v", summary)
				}
				// A second request in the same month must use the cached result.
				cached, err := svc.getSummaryAt(now)
				if err != nil || cached != summary {
					t.Fatalf("same-month cache = %+v, error = %v", cached, err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDonationSummaryBalanceAcrossSources(t *testing.T) {
	for _, source := range []string{"snapshot", "latest snapshot", "live", "stale snapshot", "no config"} {
		t.Run(source, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			svc := NewDonationService(repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock")), cache.New(time.Minute, time.Minute))
			if source == "stale snapshot" {
				svc.MarkSnapshotStale()
			} else {
				rows := sqlmock.NewRows([]string{"DS_TOTAL", "DS_DATE"})
				if source == "snapshot" {
					rows.AddRow(100000, "2026-09-23")
				}
				mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*WHERE DS_DATE`).WillReturnRows(rows)
				if source != "snapshot" {
					rows = sqlmock.NewRows([]string{"DS_TOTAL", "DS_DATE"})
					if source == "latest snapshot" {
						rows.AddRow(100000, "2026-08-31")
					}
					mock.ExpectQuery(`(?s)FROM DONATION_SNAPSHOT.*ORDER BY DS_DATE`).WillReturnRows(rows)
				}
			}
			if source == "live" || source == "stale snapshot" || source == "no config" {
				mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*COUNT\(DISTINCT CASE`).WillReturnRows(sqlmock.NewRows([]string{"TOTAL_AMOUNT", "DONOR_COUNT"}).AddRow(100000, 2))
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema.TABLES`).WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))
			}
			configRows := sqlmock.NewRows([]string{"DC_BALANCE_AMOUNT", "DC_BALANCE_AS_OF"})
			if source != "no config" {
				configRows.AddRow(int64(5000000000), "2026-09-22")
			}
			mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(configRows)
			mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_DONATION_DATE`).WithArgs("2026-09-01", "2026-10-01").
				WillReturnRows(sqlmock.NewRows([]string{"AMOUNT"}).AddRow(30000))
			summary, err := svc.getSummaryAt(time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			if summary.MonthAmount != 30000 || summary.DisplayAmount != 100000 {
				t.Fatalf("summary = %+v", summary)
			}
			if source == "no config" {
				if summary.BalanceAmount != nil || summary.BalanceAsOf != nil {
					t.Fatalf("missing config balance = %+v", summary)
				}
			} else if summary.BalanceAmount == nil || *summary.BalanceAmount != 5000000000 || summary.BalanceAsOf == nil || *summary.BalanceAsOf != "2026-09-22" {
				t.Fatalf("configured balance = %+v", summary)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDonationSummaryDoesNotCacheFailedMonthQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := cache.New(time.Minute, time.Minute)
	svc := NewDonationService(repository.NewDonationRepository(sqlx.NewDb(db, "sqlmock")), store)
	mock.ExpectQuery(`FROM DONATION_SNAPSHOT`).WillReturnRows(sqlmock.NewRows([]string{"DS_TOTAL"}).AddRow(100000))
	mock.ExpectQuery(`FROM DONATION_CONFIG`).WillReturnRows(sqlmock.NewRows([]string{"DC_BALANCE_AMOUNT"}).AddRow(nil))
	wantErr := errors.New("month query failed")
	mock.ExpectQuery(`(?s)SUM\(O_NET_RECEIVED_AMOUNT\).*O_DONATION_DATE`).WillReturnError(wantErr)
	if _, err := svc.GetSummary(); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if _, found := store.Get("donation_summary"); found {
		t.Fatal("failed summary was cached")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
