// gate.go — File-sentinel state for the maintenance write freeze.
package maintenance

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SmokeProofHeader carries an operator-controlled proof for narrow smoke writes.
const SmokeProofHeader = "X-Maintenance-Smoke-Proof"

const ProductionSentinelPath = "/run/alumni/maintenance"

// ErrWritesFrozen indicates that maintenance has rejected a direct write operation.
var ErrWritesFrozen = errors.New("writes are frozen for maintenance")

// Gate reports whether the configured maintenance sentinel is active.
type Gate struct {
	sentinelPath        string
	smokeProofHash      [sha256.Size]byte
	smokeProofEnabled   bool
	smokeAllowedPaths   map[string]struct{}
	mu                  sync.Mutex
	admissionsClosed    bool
	inFlight            int
	drained             chan struct{}
	releaseBridgePath   string
	releaseProofHash    [sha256.Size]byte
	releaseProofEnabled bool
	releaseOwnerUID     int
}

// NewGate creates a maintenance gate. An empty sentinel path keeps it disabled.
func NewGate(sentinelPath, smokeProofHash string, smokeAllowedPaths ...string) (*Gate, error) {
	gate := &Gate{
		sentinelPath:      sentinelPath,
		smokeAllowedPaths: make(map[string]struct{}, len(smokeAllowedPaths)),
		releaseOwnerUID:   -1,
	}
	for _, path := range smokeAllowedPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("maintenance smoke path must start with /")
		}
		gate.smokeAllowedPaths[path] = struct{}{}
	}
	if smokeProofHash == "" {
		return gate, nil
	}
	decoded, err := hex.DecodeString(smokeProofHash)
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("maintenance smoke proof hash must be a SHA-256 hex digest")
	}
	copy(gate.smokeProofHash[:], decoded)
	gate.smokeProofEnabled = true
	return gate, nil
}

func NewRuntimeGate(environment, sentinelPath, smokeProofHash string, smokeAllowedPaths ...string) (*Gate, error) {
	if !filepath.IsAbs(sentinelPath) {
		return nil, fmt.Errorf("maintenance sentinel path must be absolute")
	}
	environment = strings.ToLower(strings.TrimSpace(environment))
	if (environment == "prod" || environment == "production") && sentinelPath != ProductionSentinelPath {
		return nil, fmt.Errorf("production maintenance sentinel path must be %s", ProductionSentinelPath)
	}
	return NewGate(sentinelPath, smokeProofHash, smokeAllowedPaths...)
}

// Active fails closed for sentinel lookup errors other than non-existence.
func (g *Gate) Active() bool {
	if g == nil {
		return false
	}
	return pathActive(g.sentinelPath) || pathActive(g.releaseBridgePath)
}

func pathActive(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

// AllowsSmoke validates an allowlisted path and raw proof without retaining or logging it.
func (g *Gate) AllowsSmoke(path, rawProof string) bool {
	if g == nil || !g.smokeProofEnabled || rawProof == "" {
		return false
	}
	if _, allowed := g.smokeAllowedPaths[path]; !allowed {
		return false
	}
	provided := sha256.Sum256([]byte(rawProof))
	return subtle.ConstantTimeCompare(provided[:], g.smokeProofHash[:]) == 1
}
