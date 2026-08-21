// release_gate_test.go — Atomic writer admission and durable release-bridge tests.
package maintenance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testReleaseGeneration = "0123456789abcdef0123456789abcdef"
	testReleaseAttempt    = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func TestCloseAndDrainBlocksNewAdmissionsUntilExistingLeaseReturns(t *testing.T) {
	gate := newReleaseTestGate(t)
	removeReleaseAuthority(t, gate)
	lease, err := gate.EnterWriter(false)
	if err != nil {
		t.Fatalf("enter initial writer: %v", err)
	}
	activateReleaseAuthority(t, gate)

	drained := make(chan error, 1)
	go func() {
		drained <- gate.CloseAndDrain(context.Background(), testReleaseGeneration, testReleaseAttempt)
	}()

	waitForReleaseGateClosed(t, gate)
	if _, err := gate.EnterWriter(true); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("new smoke writer error = %v, want ErrWritesFrozen", err)
	}
	if _, err := gate.EnterBackground(); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("new background writer error = %v, want ErrWritesFrozen", err)
	}
	select {
	case err := <-drained:
		t.Fatalf("drain returned before existing lease: %v", err)
	default:
	}

	lease.Release()
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("close and drain: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("drain did not finish after existing lease returned")
	}
}

func TestCloseAndDrainTimeoutKeepsAdmissionsClosed(t *testing.T) {
	gate := newReleaseTestGate(t)
	removeReleaseAuthority(t, gate)
	lease, err := gate.EnterWriter(false)
	if err != nil {
		t.Fatalf("enter initial writer: %v", err)
	}
	defer lease.Release()
	activateReleaseAuthority(t, gate)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := gate.CloseAndDrain(ctx, testReleaseGeneration, testReleaseAttempt); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close and drain error = %v, want deadline exceeded", err)
	}
	if _, err := gate.EnterWriter(true); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("admission after timeout error = %v, want ErrWritesFrozen", err)
	}
}

func TestArmOpenRequiresCanonicalRemovedAndExactBridge(t *testing.T) {
	gate := newReleaseTestGate(t)
	if err := gate.CloseAndDrain(context.Background(), testReleaseGeneration, testReleaseAttempt); err != nil {
		t.Fatalf("close and drain: %v", err)
	}

	if err := gate.ArmOpen(testReleaseGeneration, testReleaseAttempt); !errors.Is(err, ErrCanonicalSentinelPresent) {
		t.Fatalf("arm with canonical sentinel error = %v, want ErrCanonicalSentinelPresent", err)
	}
	if err := os.Remove(gate.sentinelPath); err != nil {
		t.Fatal(err)
	}
	if err := gate.ArmOpen(testReleaseGeneration, testReleaseAttempt); !errors.Is(err, ErrReleaseBridgeInvalid) {
		t.Fatalf("arm with prepared bridge error = %v, want ErrReleaseBridgeInvalid", err)
	}
	writeReleaseFile(t, gate.releaseBridgePath, drainedBridgeContent(testReleaseGeneration, testReleaseAttempt))
	if err := gate.ArmOpen(testReleaseGeneration, testReleaseAttempt); err != nil {
		t.Fatalf("arm open: %v", err)
	}
	if _, err := gate.EnterWriter(false); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("bridge did not keep armed gate blocked: %v", err)
	}
	if err := os.Remove(gate.releaseBridgePath); err != nil {
		t.Fatal(err)
	}
	lease, err := gate.EnterWriter(false)
	if err != nil {
		t.Fatalf("writer did not open after bridge removal: %v", err)
	}
	lease.Release()
}

func TestArmOpenAfterProcessRestartUsesDurableDrainedBridge(t *testing.T) {
	original := newReleaseTestGate(t)
	if err := original.CloseAndDrain(context.Background(), testReleaseGeneration, testReleaseAttempt); err != nil {
		t.Fatalf("close and drain: %v", err)
	}
	if err := os.Remove(original.sentinelPath); err != nil {
		t.Fatal(err)
	}
	writeReleaseFile(t, original.releaseBridgePath, drainedBridgeContent(testReleaseGeneration, testReleaseAttempt))

	digest := sha256.Sum256([]byte("fixture-release-proof"))
	restarted, err := NewGate(original.sentinelPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.ConfigureRelease(ReleaseConfig{
		BridgePath:       original.releaseBridgePath,
		ProofSHA256:      hex.EncodeToString(digest[:]),
		ExpectedOwnerUID: os.Getuid(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := restarted.ArmOpen(testReleaseGeneration, testReleaseAttempt); err != nil {
		t.Fatalf("fresh process arm: %v", err)
	}
	if _, err := restarted.EnterBackground(); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("drained bridge did not block fresh process after arm: %v", err)
	}
}

func TestCloseAndDrainRejectsForeignBridgeAndStaysClosed(t *testing.T) {
	gate := newReleaseTestGate(t)
	foreignAttempt := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	writeReleaseFile(t, gate.releaseBridgePath, preparedBridgeContent(testReleaseGeneration, foreignAttempt))

	err := gate.CloseAndDrain(context.Background(), testReleaseGeneration, testReleaseAttempt)
	if !errors.Is(err, ErrReleaseBridgeInvalid) {
		t.Fatalf("foreign bridge error = %v, want ErrReleaseBridgeInvalid", err)
	}
	if _, err := gate.EnterWriter(true); !errors.Is(err, ErrWritesFrozen) {
		t.Fatalf("foreign bridge failure reopened admissions: %v", err)
	}
}

func TestGateActiveIncludesReleaseBridgeAndInspectionErrors(t *testing.T) {
	dir := t.TempDir()
	gate, err := NewGate(filepath.Join(dir, "maintenance"), "")
	if err != nil {
		t.Fatal(err)
	}
	gate.releaseBridgePath = filepath.Join(dir, "bridge")
	writeReleaseFile(t, gate.releaseBridgePath, preparedBridgeContent(testReleaseGeneration, testReleaseAttempt))
	if !gate.Active() {
		t.Fatal("release bridge was not treated as active maintenance")
	}
	gate.releaseBridgePath = "\x00"
	if !gate.Active() {
		t.Fatal("bridge inspection error was not fail-closed")
	}
}

func TestGateAllowsOnlyExactReleaseControlPathAndProof(t *testing.T) {
	gate := newReleaseTestGate(t)
	if !gate.AllowsRelease(ReleaseDrainPath, "fixture-release-proof") {
		t.Fatal("exact drain path and proof were rejected")
	}
	if !gate.AllowsRelease(ReleaseArmOpenPath, "fixture-release-proof") {
		t.Fatal("exact arm-open path and proof were rejected")
	}
	if gate.AllowsRelease(ReleaseDrainPath+"/extra", "fixture-release-proof") {
		t.Fatal("prefix release path was allowed")
	}
	if gate.AllowsRelease(ReleaseDrainPath, "wrong-proof") {
		t.Fatal("invalid release proof was allowed")
	}
}

func newReleaseTestGate(t *testing.T) *Gate {
	t.Helper()
	dir := t.TempDir()
	sentinelPath := filepath.Join(dir, "maintenance")
	bridgePath := filepath.Join(dir, "bridge")
	writeReleaseFile(t, sentinelPath, activeContent(testReleaseGeneration))
	writeReleaseFile(t, bridgePath, preparedBridgeContent(testReleaseGeneration, testReleaseAttempt))
	digest := sha256.Sum256([]byte("fixture-release-proof"))
	gate, err := NewGate(sentinelPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.ConfigureRelease(ReleaseConfig{
		BridgePath:       bridgePath,
		ProofSHA256:      hex.EncodeToString(digest[:]),
		ExpectedOwnerUID: os.Getuid(),
	}); err != nil {
		t.Fatalf("configure release: %v", err)
	}
	return gate
}

func waitForReleaseGateClosed(t *testing.T, gate *Gate) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !gate.AdmissionsClosed() {
		if time.Now().After(deadline) {
			t.Fatal("release gate did not close admissions")
		}
		time.Sleep(time.Millisecond)
	}
}

func writeReleaseFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeReleaseAuthority(t *testing.T, gate *Gate) {
	t.Helper()
	if err := os.Remove(gate.sentinelPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(gate.releaseBridgePath); err != nil {
		t.Fatal(err)
	}
}

func activateReleaseAuthority(t *testing.T, gate *Gate) {
	t.Helper()
	writeReleaseFile(t, gate.sentinelPath, activeContent(testReleaseGeneration))
	writeReleaseFile(t, gate.releaseBridgePath, preparedBridgeContent(testReleaseGeneration, testReleaseAttempt))
}

func activeContent(generation string) string {
	return fmt.Sprintf("state=active\ngeneration=%s\n", generation)
}

func preparedBridgeContent(generation, attempt string) string {
	return fmt.Sprintf("state=prepared\ngeneration=%s\napproval_attempt_id=%s\n", generation, attempt)
}

func drainedBridgeContent(generation, attempt string) string {
	return fmt.Sprintf("state=drained\ngeneration=%s\napproval_attempt_id=%s\n", generation, attempt)
}
