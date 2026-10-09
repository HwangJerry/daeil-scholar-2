package service

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

func TestDeferredLogoutValidatesProofBeforeDependencies(t *testing.T) {
	for _, kind := range []string{"tampered", "wrong-type", "wrong-issuer", "wrong-audience", "wrong-version", "missing-expiry", "future-issued", "bad-account", "bad-sid", "bad-jti", "wrong-signer", "none-algorithm", "expired"} {
		t.Run(kind, func(t *testing.T) {
			auth, mock, cleanup := newAuthServiceForTest(t)
			defer cleanup()
			now := time.Now()
			base := jwt.MapClaims{"iss": mobileTokenIssuer, "aud": mobileTokenAudience, "sub": "42", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nbf": now.Unix(), "typ": "refresh", "ver": 1, "sid": strings.Repeat("a", 32), "jti": strings.Repeat("b", 32)}
			method := jwt.SigningMethod(jwt.SigningMethodHS256)
			key := any([]byte(auth.cfg.JWT.Secret))
			switch kind {
			case "wrong-type":
				base["typ"] = "access"
			case "wrong-issuer":
				base["iss"] = "other"
			case "wrong-audience":
				base["aud"] = "other"
			case "wrong-version":
				base["ver"] = 2
			case "missing-expiry":
				delete(base, "exp")
			case "future-issued":
				base["iat"] = now.Add(time.Hour).Unix()
			case "bad-account":
				base["sub"] = "0"
			case "bad-sid":
				base["sid"] = "external-sid"
			case "bad-jti":
				base["jti"] = "external-jti"
			case "wrong-signer":
				key = []byte("wrong")
			case "none-algorithm":
				method = jwt.SigningMethodNone
				key = jwt.UnsafeAllowNoneSignatureType
			case "expired":
				base["iat"] = now.Add(-2 * time.Hour).Unix()
				base["nbf"] = base["iat"]
				base["exp"] = now.Add(-time.Hour).Unix()
			}
			token, err := jwt.NewWithClaims(method, base).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "tampered" {
				token += "A"
			}
			revoker, ok := any(auth).(interface {
				RevokeEndedMobileSession(context.Context, string) error
			})
			if !ok {
				t.Fatal("ended-session revocation missing")
			}
			err = revoker.RevokeEndedMobileSession(context.Background(), token)
			if kind == "expired" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, repository.ErrRefreshTokenInvalid) {
				t.Fatalf("invalid proof accepted: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeferredLogoutExistingIssuerProofIsAccepted(t *testing.T) {
	auth, mock, cleanup := newAuthServiceForTest(t)
	defer cleanup()
	token, jti, expiry, err := auth.generateMobileRefreshToken(&model.AuthUser{USRSeq: 42}, strings.Repeat("a", 32), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_SEQ,MRT_SID,EXPIRES_AT,REVOKED_AT,MRT_REVOKED_AT`).WithArgs(jti).WillReturnRows(sqlmock.NewRows([]string{"USR_SEQ", "MRT_SID", "EXPIRES_AT", "REVOKED_AT", "MRT_REVOKED_AT"}).AddRow(42, strings.Repeat("a", 32), expiry, nil, nil))
	mock.ExpectExec(`UPDATE ALUMNI_MOBILE_REFRESH_TOKEN SET REVOKED_AT`).WithArgs(42, strings.Repeat("a", 32)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := auth.RevokeEndedMobileSession(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
