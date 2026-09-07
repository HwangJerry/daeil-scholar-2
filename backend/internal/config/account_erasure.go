// account_erasure.go — Explicit rollout gates and erasure configuration validation.
package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func (c AccountErasureConfig) Validate() error {
	if c.TestUserSeq < 0 {
		return fmt.Errorf("ACCOUNT_ERASURE_TEST_USER_SEQ must be a valid non-negative integer")
	}
	if !c.RequestsEnabled && !c.WorkerEnabled {
		return nil
	}
	for name, value := range map[string]string{"ACCOUNT_ERASURE_CONTEXT_KEY": c.ContextKey, "DONATION_ARCHIVE_KEY": c.ArchiveKey} {
		key, err := hex.DecodeString(value)
		if err != nil || len(key) != 32 {
			return fmt.Errorf("%s must be a 32-byte hex key", name)
		}
	}
	if strings.EqualFold(c.ContextKey, c.ArchiveKey) {
		return fmt.Errorf("erasure context and archive keys must be distinct")
	}
	if c.LegacyRoot == "" {
		return fmt.Errorf("ACCOUNT_ERASURE_LEGACY_ROOT is required")
	}
	switch c.ExternalMode {
	case "manual": // Existing operator evidence workflow; no fake processor success.
	case "http":
		u, err := url.Parse(c.ExternalURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || c.ExternalToken == "" {
			return fmt.Errorf("external erasure requires an HTTPS endpoint and token")
		}
	default:
		return fmt.Errorf("ACCOUNT_ERASURE_EXTERNAL_MODE must be manual or http")
	}
	return nil
}

// Invalid explicit scope must fail validation, never fall back to all accounts.
func erasureTestUserFromEnv() int {
	raw, present := os.LookupEnv("ACCOUNT_ERASURE_TEST_USER_SEQ")
	if !present {
		return 0
	}
	user, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || user < 0 {
		return -1
	}
	return user
}
