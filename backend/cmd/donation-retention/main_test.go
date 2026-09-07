// main_test.go — Offline review validation must not accept an unversioned decision.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRunRequiresSourceFingerprint(t *testing.T) {
	for _, fingerprint := range []string{"", "abc", strings.Repeat("z", 64)} {
		t.Run(fingerprint, func(t *testing.T) {
			input := []map[string]interface{}{{"orderId": 123, "basis": "erase", "evidenceReference": "synthetic-review", "sourceFingerprint": fingerprint}}
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(t.TempDir(), "review.json")
			if err = os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err = run(file, false); err == nil || !strings.Contains(err.Error(), "sourceFingerprint") {
				t.Fatalf("invalid source snapshot accepted: %v", err)
			}
		})
	}
}

func TestDryRunChecksFormatWithoutDatabaseConnection(t *testing.T) {
	t.Setenv("DB_HOST", "invalid.example")
	file := filepath.Join(t.TempDir(), "review.json")
	data := `[{"orderId":123,"basis":"erase","evidenceReference":"synthetic-review","sourceFingerprint":"` + strings.Repeat("ab", 32) + `"}]`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(file, false); err != nil {
		t.Fatal(err)
	}
}
