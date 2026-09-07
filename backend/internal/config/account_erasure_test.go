// account_erasure_test.go — Rollout gates never accept partial active configuration.
package config

import (
	"strings"
	"testing"
)

func TestErasureRolloutRequiresExplicitReadiness(t *testing.T) {
	c := AccountErasureConfig{}
	if c.Validate() != nil {
		t.Fatal("paused rollout must boot")
	}
	c.RequestsEnabled = true
	if c.Validate() == nil {
		t.Fatal("active requests accepted without keys")
	}
	c.ContextKey = strings.Repeat("ab", 32)
	c.ArchiveKey = strings.Repeat("cd", 32)
	c.LegacyRoot = "/verified/root"
	if c.Validate() == nil {
		t.Fatal("missing external workflow accepted")
	}
	c.ExternalMode = "manual"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.ExternalMode = "http"
	if c.Validate() == nil {
		t.Fatal("missing HTTP processor accepted")
	}
	c.ExternalURL = "https://processor.example.org/erase"
	c.ExternalToken = "test-only"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.ArchiveKey = c.ContextKey
	if c.Validate() == nil {
		t.Fatal("key reuse accepted")
	}
}
