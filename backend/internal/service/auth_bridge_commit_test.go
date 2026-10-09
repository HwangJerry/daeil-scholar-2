package service

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBridgeLoginDatabaseFailureCannotIssueAuthenticationCookies(t *testing.T) {
	for _, stage := range []string{"login-log", "last-login"} {
		t.Run(stage, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			auth := NewAuthService(repository.NewAuthRepository(sqlx.NewDb(db, "sqlmock")), nil, &config.Config{JWT: config.JWTConfig{Secret: "synthetic", MaxAge: time.Hour}}, cache.New(time.Minute, time.Minute), zerolog.Nop())
			if stage == "login-log" {
				mock.ExpectExec(`INSERT INTO WEO_MEMBER_LOG`).WillReturnError(errors.New("synthetic log fault"))
			} else {
				mock.ExpectExec(`INSERT INTO WEO_MEMBER_LOG`).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(`UPDATE WEO_MEMBER`).WillReturnError(errors.New("synthetic last-login fault"))
			}
			w := httptest.NewRecorder()
			err = auth.LoginWithBridge(&model.User{USRSeq: 42, USRID: "synthetic", USRStatus: "CCC"}, w, httptest.NewRequest("POST", "/api/auth/social/link", nil))
			if err == nil {
				t.Fatal("fault did not fail login")
			}
			if cookies := w.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("failure issued %d cookies", len(cookies))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
