// release_gate.go — Atomic writer admission and durable maintenance release handoff.
package maintenance

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
)

const (
	ReleaseProofHeader          = "X-Maintenance-Release-Proof"
	ReleaseDrainPath            = "/internal/maintenance/drain"
	ReleaseArmOpenPath          = "/internal/maintenance/arm-open"
	ProductionReleaseBridgePath = "/run/alumni/maintenance-release-bridge"
)

var (
	ErrReleaseBridgeInvalid       = errors.New("maintenance release bridge is invalid")
	ErrCanonicalSentinelInvalid   = errors.New("canonical maintenance sentinel is invalid")
	ErrCanonicalSentinelPresent   = errors.New("canonical maintenance sentinel is still present")
	errReleaseGateStillDraining   = errors.New("maintenance release admissions are still draining")
	releaseGenerationPattern      = regexp.MustCompile(`^[a-f0-9]{32}$`)
	releaseApprovalAttemptPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type ReleaseConfig struct {
	BridgePath       string
	ProofSHA256      string
	ExpectedOwnerUID int
}

type Lease struct {
	gate *Gate
	once sync.Once
}

func (l *Lease) Release() {
	if l == nil || l.gate == nil {
		return
	}
	l.once.Do(l.gate.releaseLease)
}

func (g *Gate) ConfigureRelease(config ReleaseConfig) error {
	if g == nil {
		return errors.New("maintenance gate is nil")
	}
	if !filepath.IsAbs(config.BridgePath) {
		return errors.New("maintenance release bridge path must be absolute")
	}
	if config.ExpectedOwnerUID < 0 {
		return errors.New("maintenance release owner UID must be non-negative")
	}
	g.releaseBridgePath = config.BridgePath
	g.releaseOwnerUID = config.ExpectedOwnerUID
	if config.ProofSHA256 == "" {
		g.releaseProofEnabled = false
		return nil
	}
	decoded, err := hex.DecodeString(config.ProofSHA256)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("maintenance release proof hash must be a SHA-256 hex digest")
	}
	copy(g.releaseProofHash[:], decoded)
	g.releaseProofEnabled = true
	return nil
}

func (g *Gate) AllowsRelease(path, rawProof string) bool {
	if g == nil || !g.releaseProofEnabled || rawProof == "" {
		return false
	}
	if path != ReleaseDrainPath && path != ReleaseArmOpenPath {
		return false
	}
	provided := sha256.Sum256([]byte(rawProof))
	return subtle.ConstantTimeCompare(provided[:], g.releaseProofHash[:]) == 1
}

func (g *Gate) EnterWriter(allowSmoke bool) (*Lease, error) {
	if g == nil {
		return &Lease{}, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.admissionsClosed || pathActive(g.releaseBridgePath) || (!allowSmoke && pathActive(g.sentinelPath)) {
		return nil, ErrWritesFrozen
	}
	g.inFlight++
	return &Lease{gate: g}, nil
}

func (g *Gate) EnterBackground() (*Lease, error) {
	return g.EnterWriter(false)
}

func (g *Gate) AdmissionsClosed() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.admissionsClosed
}

func (g *Gate) CloseAndDrain(ctx context.Context, generation, attempt string) error {
	if g == nil {
		return errors.New("maintenance gate is nil")
	}
	g.closeAdmissions()
	if err := g.validateReleaseAuthority(generation, attempt, true); err != nil {
		return err
	}
	g.mu.Lock()
	drained := g.drained
	g.mu.Unlock()
	select {
	case <-drained:
	case <-ctx.Done():
		return ctx.Err()
	}
	return g.validateReleaseAuthority(generation, attempt, true)
}

func (g *Gate) ArmOpen(generation, attempt string) error {
	if g == nil {
		return errors.New("maintenance gate is nil")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.inFlight != 0 {
		return errReleaseGateStillDraining
	}
	if _, err := os.Lstat(g.sentinelPath); err == nil {
		return ErrCanonicalSentinelPresent
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: canonical sentinel lookup", ErrCanonicalSentinelInvalid)
	}
	if err := g.validateReleaseFile(g.releaseBridgePath, drainedBridgeReleaseContent(generation, attempt), ErrReleaseBridgeInvalid); err != nil {
		return err
	}
	g.admissionsClosed = false
	g.drained = nil
	return nil
}

func (g *Gate) closeAdmissions() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.admissionsClosed {
		return
	}
	g.admissionsClosed = true
	g.drained = make(chan struct{})
	if g.inFlight == 0 {
		close(g.drained)
	}
}

func (g *Gate) releaseLease() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.inFlight == 0 {
		return
	}
	g.inFlight--
	if g.admissionsClosed && g.inFlight == 0 && g.drained != nil {
		close(g.drained)
	}
}

func (g *Gate) validateReleaseAuthority(generation, attempt string, requireCanonical bool) error {
	if !releaseGenerationPattern.MatchString(generation) || !releaseApprovalAttemptPattern.MatchString(attempt) {
		return ErrReleaseBridgeInvalid
	}
	if requireCanonical {
		if err := g.validateReleaseFile(g.sentinelPath, activeReleaseContent(generation), ErrCanonicalSentinelInvalid); err != nil {
			return err
		}
	}
	return g.validateReleaseFile(g.releaseBridgePath, preparedBridgeReleaseContent(generation, attempt), ErrReleaseBridgeInvalid)
}

func (g *Gate) validateReleaseFile(path, expected string, authorityError error) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o644 {
		return authorityError
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != g.releaseOwnerUID {
		return authorityError
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != expected {
		return authorityError
	}
	return nil
}

func activeReleaseContent(generation string) string {
	return fmt.Sprintf("state=active\ngeneration=%s\n", generation)
}

func preparedBridgeReleaseContent(generation, attempt string) string {
	return fmt.Sprintf("state=prepared\ngeneration=%s\napproval_attempt_id=%s\n", generation, attempt)
}

func drainedBridgeReleaseContent(generation, attempt string) string {
	return fmt.Sprintf("state=drained\ngeneration=%s\napproval_attempt_id=%s\n", generation, attempt)
}
