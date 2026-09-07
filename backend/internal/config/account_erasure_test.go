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

func TestInvalidErasureTestUserNeverBroadensScope(t *testing.T) {
	for _, raw := range []string{"42x", "", "9999999999999999999999999", "-1"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("ACCOUNT_ERASURE_TEST_USER_SEQ", raw)
			c := Load().AccountErasure
			if c.TestUserSeq >= 0 || c.Validate() == nil {
				t.Fatalf("invalid test user %q accepted as scope %d", raw, c.TestUserSeq)
			}
		})
	}
	for raw, expected := range map[string]int{"0": 0, "42": 42} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("ACCOUNT_ERASURE_TEST_USER_SEQ", raw)
			if got := Load().AccountErasure.TestUserSeq; got != expected {
				t.Fatalf("scope changed: got %d, want %d", got, expected)
			}
		})
	}
}

func TestTestUserScopeRejectsGlobalRetention(t *testing.T) {
	for _, user := range []int{0, 42} {
		for flags := 0; flags < 8; flags++ {
			c := AccountErasureConfig{
				TestUserSeq: user, RequestsEnabled: flags&1 != 0, WorkerEnabled: flags&2 != 0,
				RetentionEnabled: flags&4 != 0, ContextKey: strings.Repeat("ab", 32),
				ArchiveKey: strings.Repeat("cd", 32), LegacyRoot: "/verified/root", ExternalMode: "manual",
			}
			err := c.Validate()
			shouldReject := user > 0 && c.RetentionEnabled
			if (err != nil) != shouldReject {
				t.Fatalf("scope %d flags %d: validation error %v", user, flags, err)
			}
		}
	}
}
