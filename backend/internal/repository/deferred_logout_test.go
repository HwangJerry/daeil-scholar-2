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
			} else if kind == "revoked" {
				mock.ExpectCommit()
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

func TestDeferredGlobalProofRepositoryRejectsReplayAndRollsBack(t *testing.T) {
	for _, kind := range []string{"missing", "revoked", "expired", "legacy-delete-failure", "push-delete-failure", "commit-failure", "success"} {
		t.Run(kind, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			expiry := time.Now().Add(time.Hour).Truncate(time.Second)
			mock.ExpectBegin()
			rows := sqlmock.NewRows([]string{"USR_SEQ", "MRT_SID", "EXPIRES_AT", "REVOKED_AT", "MRT_REVOKED_AT"})
			var revoked any
			if kind == "revoked" {
				revoked = time.Now()
			}
			if kind == "expired" {
				expiry = time.Now().Add(-time.Hour).Truncate(time.Second)
			}
			if kind != "missing" {
				rows.AddRow(42, "family", expiry, revoked, nil)
			}
			mock.ExpectQuery(`SELECT USR_SEQ,MRT_SID,EXPIRES_AT,REVOKED_AT,MRT_REVOKED_AT.*FOR UPDATE`).WithArgs("proof").WillReturnRows(rows)
			invalid := kind == "missing" || kind == "revoked" || kind == "expired"
			if invalid {
				mock.ExpectRollback()
			} else {
				mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN.*WHERE USR_SEQ=\?`).WithArgs(42).WillReturnResult(sqlmock.NewResult(0, 3))
				legacy := mock.ExpectExec(`DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ=\?`).WithArgs(42)
				if kind == "legacy-delete-failure" {
					legacy.WillReturnError(errors.New("legacy delete"))
					mock.ExpectRollback()
				} else {
					legacy.WillReturnResult(sqlmock.NewResult(0, 1))
					push := mock.ExpectExec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ=\?`).WithArgs(42)
					if kind == "push-delete-failure" {
						push.WillReturnError(errors.New("push delete"))
						mock.ExpectRollback()
					} else {
						push.WillReturnResult(sqlmock.NewResult(0, 2))
						if kind == "commit-failure" {
							mock.ExpectCommit().WillReturnError(errors.New("commit"))
						} else {
							mock.ExpectCommit()
						}
					}
				}
			}
			err = repo.RevokeAllSessionsByProof(context.Background(), 42, "family", "proof", expiry)
			if invalid && !errors.Is(err, ErrRefreshTokenInvalid) {
				t.Fatalf("invalid error=%v", err)
			}
			if kind == "success" && err != nil {
				t.Fatal(err)
			}
			if kind != "success" && err == nil {
				t.Fatal("failure suppressed")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeferredLogoutResultRequiresRetainedProofAndCommit(t *testing.T) {
	for _, kind := range []string{"missing", "expired", "revoked-confirmed", "revoked-commit-failure", "query-failure"} {
		t.Run(kind, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := NewAuthRepository(sqlx.NewDb(db, "sqlmock"))
			expiry := time.Now().Add(time.Hour).Truncate(time.Second)
			if kind == "expired" {
				expiry = time.Now().Add(-time.Minute).Truncate(time.Second)
			}
			mock.ExpectBegin()
			q := mock.ExpectQuery(`SELECT USR_SEQ,MRT_SID,EXPIRES_AT,REVOKED_AT,MRT_REVOKED_AT.*FOR UPDATE`).WithArgs("proof")
			if kind == "query-failure" {
				q.WillReturnError(errors.New("synthetic query"))
			} else {
				rows := sqlmock.NewRows([]string{"USR_SEQ", "MRT_SID", "EXPIRES_AT", "REVOKED_AT", "MRT_REVOKED_AT"})
				if kind != "missing" {
					rows.AddRow(42, "family", expiry, time.Now(), nil)
				}
				q.WillReturnRows(rows)
			}
			if kind == "revoked-confirmed" {
				mock.ExpectCommit()
			} else if kind == "revoked-commit-failure" {
				mock.ExpectCommit().WillReturnError(errors.New("synthetic commit"))
			} else {
				mock.ExpectRollback()
			}
			confirmed, err := repo.RevokeMobileSessionByProofWithDeviceResult(context.Background(), 42, "family", "proof", expiry, "")
			if confirmed != (kind == "revoked-confirmed") {
				t.Fatalf("confirmed=%v kind=%s", confirmed, kind)
			}
			expectError := kind == "query-failure" || kind == "revoked-commit-failure"
			if (err != nil) != expectError {
				t.Fatalf("kind=%s error=%v", kind, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
