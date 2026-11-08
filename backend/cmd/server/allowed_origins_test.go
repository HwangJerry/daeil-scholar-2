package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dflh-saf/backend/internal/config"
	mw "github.com/dflh-saf/backend/internal/middleware"
)

func TestConfiguredOriginsAddsAdminOrigin(t *testing.T) {
	got := configuredOrigins(config.ServerConfig{
		AllowedOrigin: "https://daeilfoundation.or.kr",
		AdminOrigin:   " https://adms.daeilfoundation.or.kr ",
	})
	want := []string{"https://daeilfoundation.or.kr", "https://adms.daeilfoundation.or.kr"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}

func TestMobileOriginAllowsWebSheetsWithoutChangingWebLoginOrigin(t *testing.T) {
	server := config.ServerConfig{
		AllowedOrigin: "https://daeilfoundation.or.kr",
		AdminOrigin:   "https://adms.daeilfoundation.or.kr",
		MobileOrigin:  " https://211.169.249.240 ",
	}
	origins := configuredOrigins(server)
	if server.AllowedOrigin != "https://daeilfoundation.or.kr" {
		t.Fatal("web login redirect origin changed")
	}
	handler := mw.CORSMiddleware(origins)(mw.CSRFMiddleware(origins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	for _, test := range []struct {
		origin string
		status int
	}{
		{"https://211.169.249.240", http.StatusNoContent},
		{"https://daeilfoundation.or.kr", http.StatusNoContent},
		{"https://adms.daeilfoundation.or.kr", http.StatusNoContent},
		{"http://211.169.249.240", http.StatusForbidden},
		{"https://211.169.249.240.evil.test", http.StatusForbidden},
		{"https://untrusted.example", http.StatusForbidden},
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/account-deletion/receipt", nil)
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Errorf("origin %q returned %d, want %d", test.origin, w.Code, test.status)
		}
		if test.status == http.StatusNoContent && w.Header().Get("Access-Control-Allow-Origin") != test.origin {
			t.Errorf("origin %q was not returned for CORS", test.origin)
		}
	}
	for _, test := range []struct {
		referer string
		status  int
	}{
		{"https://211.169.249.240/account-deletion", http.StatusNoContent},
		{"https://211.169.249.240.evil.test/account-deletion", http.StatusForbidden},
		{"https://211.169.249.240@evil.test/account-deletion", http.StatusForbidden},
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/account-deletion/receipt", nil)
		r.Header.Set("Referer", test.referer)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Errorf("referer %q returned %d, want %d", test.referer, w.Code, test.status)
		}
	}
}

func TestConfiguredOriginsWithoutAdminOrigin(t *testing.T) {
	got := configuredOrigins(config.ServerConfig{AllowedOrigin: "https://a.example, https://b.example"})
	want := []string{"https://a.example", "https://b.example"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
}
