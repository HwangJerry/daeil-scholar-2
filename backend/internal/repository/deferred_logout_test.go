package repository

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"testing"
	"time"
)

func TestDeferredLogoutRepositoryBindingAndAtomicFailure(t *testing.T) {
	for _, kind := range []string{"consumed-ancestor", "missing", "revoked", "expired", "wrong-account", "wrong-sid", "wrong-expiry", "query-failure", "update-failure", "commit-failure", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			expiry := time.Now().Add(time.Hour).Truncate(time.Second)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
				if err := repo.RevokeMobileSessionByProof(ctx, 42, "family", "proof", expiry); !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				return
			}
			mock.ExpectBegin()
			q := mock.ExpectQuery(`SELECT USR_SEQ,MRT_SID,EXPIRES_AT,REVOKED_AT,MRT_REVOKED_AT`).WithArgs("proof")
			if kind == "query-failure" {
				q.WillReturnError(errors.New("synthetic query"))
			} else {
				rows := sqlmock.NewRows([]string{"USR_SEQ", "MRT_SID", "EXPIRES_AT", "REVOKED_AT", "MRT_REVOKED_AT"})
				account, sid, rowExpiry := 42, "family", expiry
				var revoked any
				switch kind {
				case "wrong-account":
					account = 43
				case "wrong-sid":
					sid = "unrelated"
				case "wrong-expiry":
					rowExpiry = expiry.Add(time.Second)
				case "revoked":
					revoked = time.Now()
				case "expired":
					rowExpiry = time.Now().Add(-time.Minute).Truncate(time.Second)
					expiry = rowExpiry
				}
				if kind != "missing" {
					rows.AddRow(account, sid, rowExpiry, revoked, nil)
				}
				q.WillReturnRows(rows)
			}
			writes := kind == "consumed-ancestor" || kind == "update-failure" || kind == "commit-failure"
			if writes {
				e := mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT`).WithArgs(42, "family")
				if kind == "update-failure" {
					e.WillReturnError(errors.New("synthetic update"))
					mock.ExpectRollback()
				} else {
					e.WillReturnResult(sqlmock.NewResult(0, 2))
					if kind == "commit-failure" {
						mock.ExpectCommit().WillReturnError(errors.New("synthetic commit"))
					} else {
						mock.ExpectCommit()
					}
				}
			} else {
				mock.ExpectRollback()
			}
			err = repo.RevokeMobileSessionByProof(ctx, 42, "family", "proof", expiry)
			switch kind {
			case "wrong-account", "wrong-sid", "wrong-expiry":
				if !errors.Is(err, ErrRefreshTokenInvalid) {
					t.Fatal(err)
				}
			case "query-failure", "update-failure", "commit-failure":
				if err == nil {
					t.Fatal("DB error suppressed")
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
