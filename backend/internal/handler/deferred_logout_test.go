package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeferredLogoutBodyBoundaryAndNoCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, body, media string
		status            int
	}{
		{"tampered", `{"refreshToken":"invalid"}`, "application/json", 401},
		{"empty", `{"refreshToken":""}`, "application/json", 400},
		{"null", `null`, "application/json", 400},
		{"broken", `{`, "application/json", 400},
		{"extra-account", `{"refreshToken":"invalid","accountId":42}`, "application/json", 400},
		{"trailing", `{"refreshToken":"invalid"}{}`, "application/json", 400},
		{"not-json", `{"refreshToken":"invalid"}`, "text/plain", 415},
		{"oversize", `{"refreshToken":"` + strings.Repeat("x", deferredLogoutBodyLimit) + `"}`, "application/json", 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, mock, cleanup := newLogoutAuthHandlerForTest(t)
			defer cleanup()
			r := httptest.NewRequest(http.MethodPost, "/api/auth/logout/deferred", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.media)
			w := httptest.NewRecorder()
			h.DeferredLogout(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d want %d", w.Code, tc.status)
			}
			if len(w.Result().Cookies()) != 0 {
				t.Fatal("revoke minted cookies")
			}
			if strings.Contains(w.Body.String(), "accessToken") {
				t.Fatal("revoke minted credentials")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
