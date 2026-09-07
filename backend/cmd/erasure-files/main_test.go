// main_test.go — Operational manifests cannot silently accept unreviewed fields.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPlanRequiresPrivateStrictJSON(t *testing.T) {
	valid := `{"requestId":1,"userSeq":42,"evidenceReference":"review-42","ownershipVerified":true,"retentionRespected":true,"files":["/files/profile/old.jpg"]}`
	file := filepath.Join(t.TempDir(), "plan.json")
	for _, body := range []string{valid + ` {}`, strings.Replace(valid, `"files":`, `"unknown":true,"files":`, 1)} {
		if err := os.WriteFile(file, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readPlan(file); err == nil {
			t.Fatal("ambiguous JSON accepted")
		}
	}
	if err := os.WriteFile(file, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPlan(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPlan(file); err == nil {
		t.Fatal("world-readable inventory accepted")
	}
}
