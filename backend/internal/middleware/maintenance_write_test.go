// maintenance_write_test.go — Maintenance-mode HTTP write blocking tests.
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
)

func TestMaintenanceWriteMiddlewareRejectsPostWhenSentinelExists(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recorder := httptest.NewRecorder()

	MaintenanceWriteMiddleware(gate)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("mutating handler was called during maintenance")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if got := recorder.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
}

func TestMaintenanceWriteMiddlewareAllowsValidSmokeProof(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawProof := "fixture-controlled-smoke-proof"
	digest := sha256.Sum256([]byte(rawProof))
	gate, err := maintenance.NewGate(sentinel, hex.EncodeToString(digest[:]), "/api/auth/login")
	if err != nil {
		t.Fatal(err)
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.RemoteAddr = "127.0.0.1:4321"
	request.Header.Set(maintenance.SmokeProofHeader, rawProof)
	recorder := httptest.NewRecorder()

	MaintenanceWriteMiddleware(gate)(next).ServeHTTP(recorder, request)

	if !called {
		t.Fatal("valid controlled-smoke request was blocked")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestMaintenanceWriteMiddlewareRejectsValidSmokeProofFromNonLoopback(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawProof := "fixture-controlled-smoke-proof"
	digest := sha256.Sum256([]byte(rawProof))
	gate, err := maintenance.NewGate(sentinel, hex.EncodeToString(digest[:]), "/api/auth/login")
	if err != nil {
		t.Fatal(err)
	}

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.RemoteAddr = "198.51.100.10:4321"
	request.Header.Set(maintenance.SmokeProofHeader, rawProof)
	recorder := httptest.NewRecorder()

	MaintenanceWriteMiddleware(gate)(next).ServeHTTP(recorder, request)

	if called {
		t.Fatal("non-loopback controlled-smoke request reached handler")
	}
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestMaintenanceWriteMiddlewareRejectsValidProofOutsideAllowlist(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawProof := "fixture-controlled-smoke-proof"
	digest := sha256.Sum256([]byte(rawProof))
	gate, err := maintenance.NewGate(sentinel, hex.EncodeToString(digest[:]), "/api/auth/login")
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/admin/feed", nil)
	request.Header.Set(maintenance.SmokeProofHeader, rawProof)
	recorder := httptest.NewRecorder()

	MaintenanceWriteMiddleware(gate)(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestMaintenanceWriteMiddlewareRejectsMutatingGetRoutes(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/api/auth/kakao/callback?code=fixture",
		"/pg/easypay/relay",
		"/api/feed/123",
		"/api/disclosure/456",
	} {
		t.Run(path, func(t *testing.T) {
			nextCalled := false
			handler := MaintenanceWriteMiddleware(gate)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusNoContent)
			}))
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
			}
			if nextCalled {
				t.Fatal("mutating GET route reached handler during maintenance")
			}
		})
	}
}

func TestMaintenanceWriteMiddlewareAllowsReadOnlyGet(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}

	handler := MaintenanceWriteMiddleware(gate)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, path := range []string{
		"/api/health",
		"/api/feed/hero",
		"/api/feed/123/siblings",
		"/api/feed/123/comments",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
		})
	}
}

func TestMaintenanceWriteMiddlewareLeaseDrainsExistingHandler(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "maintenance")
	bridge := filepath.Join(dir, "bridge")
	rawProof := "fixture-release-proof"
	digest := sha256.Sum256([]byte(rawProof))
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.ConfigureRelease(maintenance.ReleaseConfig{
		BridgePath:       bridge,
		ProofSHA256:      hex.EncodeToString(digest[:]),
		ExpectedOwnerUID: os.Getuid(),
	}); err != nil {
		t.Fatal(err)
	}

	handlerStarted := make(chan struct{})
	handlerRelease := make(chan struct{})
	requestDone := make(chan struct{})
	handler := MaintenanceWriteMiddleware(gate)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(handlerStarted)
		<-handlerRelease
		w.WriteHeader(http.StatusNoContent)
	}))
	go func() {
		defer close(requestDone)
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}()
	<-handlerStarted

	generation := "0123456789abcdef0123456789abcdef"
	attempt := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	writeMaintenanceAuthority(t, sentinel, "state=active\ngeneration="+generation+"\n")
	writeMaintenanceAuthority(t, bridge, "state=prepared\ngeneration="+generation+"\napproval_attempt_id="+attempt+"\n")
	drained := make(chan error, 1)
	go func() { drained <- gate.CloseAndDrain(context.Background(), generation, attempt) }()
	waitForAdmissionsClosed(t, gate)
	select {
	case err := <-drained:
		t.Fatalf("drain returned before HTTP handler completed: %v", err)
	default:
	}

	close(handlerRelease)
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("HTTP handler did not complete")
	}
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("drain: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("drain did not observe released HTTP lease")
	}
}

func writeMaintenanceAuthority(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
}

func waitForAdmissionsClosed(t *testing.T, gate *maintenance.Gate) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !gate.AdmissionsClosed() {
		if time.Now().After(deadline) {
			t.Fatal("admissions did not close")
		}
		time.Sleep(time.Millisecond)
	}
}
