// routes_maintenance_test.go — Root-router maintenance middleware wiring tests.
package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

func TestRegisterRoutesBlocksMutatingRequestsDuringMaintenance(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	router := registerRoutes(
		handlers{}, nil, cache.New(time.Minute, time.Minute), nil,
		&config.Config{}, zerolog.Nop(), nil, gate,
	)
	request := httptest.NewRequest(http.MethodPost, "/api/unknown-write", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
