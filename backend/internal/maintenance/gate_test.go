// gate_test.go — Fail-closed sentinel and controlled-smoke authorization tests.
package maintenance

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNewRuntimeGateRejectsWrongProductionSentinel(t *testing.T) {
	if _, err := NewRuntimeGate("prod", "/tmp/maintenance", ""); err == nil {
		t.Fatal("production runtime accepted a non-canonical maintenance sentinel")
	}
}

func TestNewRuntimeGateAcceptsCanonicalProductionSentinel(t *testing.T) {
	if _, err := NewRuntimeGate("prod", "/run/alumni/maintenance", ""); err != nil {
		t.Fatalf("canonical production sentinel: %v", err)
	}
}

func TestGateFailsClosedWhenSentinelCannotBeInspected(t *testing.T) {
	gate, err := NewGate("\x00", "")
	if err != nil {
		t.Fatal(err)
	}
	if !gate.Active() {
		t.Fatal("sentinel inspection error was treated as inactive maintenance")
	}
}

func TestGateAllowsOnlyExactConfiguredSmokePathAndProof(t *testing.T) {
	rawProof := "fixture-proof"
	digest := sha256.Sum256([]byte(rawProof))
	gate, err := NewGate("", hex.EncodeToString(digest[:]), "/api/auth/login")
	if err != nil {
		t.Fatal(err)
	}

	if !gate.AllowsSmoke("/api/auth/login", rawProof) {
		t.Fatal("exact configured smoke request was rejected")
	}
	if gate.AllowsSmoke("/api/auth/logout", rawProof) {
		t.Fatal("unconfigured smoke path was allowed")
	}
	if gate.AllowsSmoke("/api/auth/login", "wrong-proof") {
		t.Fatal("invalid smoke proof was allowed")
	}
}

func TestNewGateRejectsInvalidSmokeConfiguration(t *testing.T) {
	for name, testCase := range map[string]struct {
		digest string
		paths  []string
	}{
		"digest": {digest: "not-a-sha256"},
		"path":   {paths: []string{"api/auth/login"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewGate("", testCase.digest, testCase.paths...); err == nil {
				t.Fatal("invalid smoke configuration was accepted")
			}
		})
	}
}
