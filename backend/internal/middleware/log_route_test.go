// log_route_test.go — Route logs must never retain IDs, tokens or panic payloads.
package middleware

import (
	"bytes"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogsUseRouteTemplates(t *testing.T) {
	var output bytes.Buffer
	router := chi.NewRouter()
	router.Use(RequestLogger(zerolog.New(&output)))
	router.Get("/api/members/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	router.Get("/uploads/*", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	for _, raw := range []string{"/api/members/987654?token=supersecret", "/uploads/private-person.jpg", "/unknown/person@example.org"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", raw, nil))
	}
	logs := output.String()
	for _, secret := range []string{"987654", "supersecret", "private-person", "person@example.org"} {
		if strings.Contains(logs, secret) {
			t.Fatalf("private URL retained: %s", secret)
		}
	}
	for _, route := range []string{"/api/members/{id}", "/uploads/*", "unmatched"} {
		if !strings.Contains(logs, route) {
			t.Fatalf("missing route metric: %s", route)
		}
	}
}

func TestPanicLogsDoNotIncludePrivatePayloadOrPath(t *testing.T) {
	var output bytes.Buffer
	router := chi.NewRouter()
	router.Use(Recoverer(zerolog.New(&output), nil))
	router.Get("/api/members/{id}", func(w http.ResponseWriter, r *http.Request) { panic("person@example.org bearer-secret") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/members/987654", nil))
	if response.Code != 500 {
		t.Fatal("panic did not return generic error")
	}
	for _, secret := range []string{"person@example.org", "bearer-secret", "987654"} {
		if strings.Contains(output.String(), secret) || strings.Contains(response.Body.String(), secret) {
			t.Fatal("private panic payload retained", secret)
		}
	}
	if !strings.Contains(output.String(), "/api/members/{id}") {
		t.Fatal("panic route template missing")
	}
}
