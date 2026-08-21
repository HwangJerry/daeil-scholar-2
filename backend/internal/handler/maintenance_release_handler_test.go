// maintenance_release_handler_test.go — Loopback release-control contract tests.
package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
)

const (
	handlerReleaseGeneration = "0123456789abcdef0123456789abcdef"
	handlerReleaseAttempt    = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	handlerReleaseProof      = "fixture-release-proof"
)

func TestMaintenanceReleaseHandlerCompletesTwoPhaseHandoff(t *testing.T) {
	gate, sentinel, bridge := newMaintenanceReleaseHandlerGate(t)
	handler := NewMaintenanceReleaseHandler(gate, time.Second)

	drain := newMaintenanceReleaseRequest(t, maintenance.ReleaseDrainPath, handlerReleaseProof)
	drainRecorder := httptest.NewRecorder()
	handler.Drain(drainRecorder, drain)
	if drainRecorder.Code != http.StatusOK || !strings.Contains(drainRecorder.Body.String(), `"state":"DRAINED"`) {
		t.Fatalf("drain response = %d %s", drainRecorder.Code, drainRecorder.Body.String())
	}
	if !gate.AdmissionsClosed() {
		t.Fatal("drain response did not leave admissions closed")
	}

	if err := os.Remove(sentinel); err != nil {
		t.Fatal(err)
	}
	writeHandlerReleaseFile(t, bridge, "state=drained\ngeneration="+handlerReleaseGeneration+"\napproval_attempt_id="+handlerReleaseAttempt+"\n")
	arm := newMaintenanceReleaseRequest(t, maintenance.ReleaseArmOpenPath, handlerReleaseProof)
	armRecorder := httptest.NewRecorder()
	handler.ArmOpen(armRecorder, arm)
	if armRecorder.Code != http.StatusOK || !strings.Contains(armRecorder.Body.String(), `"state":"ARMED"`) {
		t.Fatalf("arm response = %d %s", armRecorder.Code, armRecorder.Body.String())
	}
	if _, err := gate.EnterWriter(false); err == nil {
		t.Fatal("bridge did not block writers after arm-open")
	}
	if err := os.Remove(bridge); err != nil {
		t.Fatal(err)
	}
	lease, err := gate.EnterWriter(false)
	if err != nil {
		t.Fatalf("writer remained blocked after bridge removal: %v", err)
	}
	lease.Release()
}

func TestMaintenanceReleaseHandlerRejectsNonLoopbackWithoutClosing(t *testing.T) {
	gate, _, _ := newMaintenanceReleaseHandlerGate(t)
	handler := NewMaintenanceReleaseHandler(gate, time.Second)
	request := newMaintenanceReleaseRequest(t, maintenance.ReleaseDrainPath, handlerReleaseProof)
	request.RemoteAddr = "198.51.100.10:4321"
	recorder := httptest.NewRecorder()

	handler.Drain(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if gate.AdmissionsClosed() {
		t.Fatal("unauthorized request closed admissions")
	}
}

func TestMaintenanceReleaseHandlerForeignAuthorityFailsClosed(t *testing.T) {
	gate, _, bridge := newMaintenanceReleaseHandlerGate(t)
	writeHandlerReleaseFile(t, bridge, "state=prepared\ngeneration="+handlerReleaseGeneration+"\napproval_attempt_id="+strings.Repeat("a", 64)+"\n")
	handler := NewMaintenanceReleaseHandler(gate, time.Second)
	request := newMaintenanceReleaseRequest(t, maintenance.ReleaseDrainPath, handlerReleaseProof)
	recorder := httptest.NewRecorder()

	handler.Drain(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	if !gate.AdmissionsClosed() {
		t.Fatal("foreign authority failure reopened admissions")
	}
}

func newMaintenanceReleaseHandlerGate(t *testing.T) (*maintenance.Gate, string, string) {
	t.Helper()
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "maintenance")
	bridge := filepath.Join(dir, "bridge")
	writeHandlerReleaseFile(t, sentinel, "state=active\ngeneration="+handlerReleaseGeneration+"\n")
	writeHandlerReleaseFile(t, bridge, "state=prepared\ngeneration="+handlerReleaseGeneration+"\napproval_attempt_id="+handlerReleaseAttempt+"\n")
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(handlerReleaseProof))
	if err := gate.ConfigureRelease(maintenance.ReleaseConfig{
		BridgePath:       bridge,
		ProofSHA256:      hex.EncodeToString(digest[:]),
		ExpectedOwnerUID: os.Getuid(),
	}); err != nil {
		t.Fatal(err)
	}
	return gate, sentinel, bridge
}

func newMaintenanceReleaseRequest(t *testing.T, path, proof string) *http.Request {
	t.Helper()
	body := `{"generation":"` + handlerReleaseGeneration + `","approval_attempt_id":"` + handlerReleaseAttempt + `"}`
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.RemoteAddr = "127.0.0.1:4321"
	request.Header.Set(maintenance.ReleaseProofHeader, proof)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func writeHandlerReleaseFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
}
