// pg_audit_privacy_test.go — Sensitive PG values never enter newly written logs.
package service

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPGAuditKeepsEvidenceWithoutRawCardOrError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	logger, err := NewPGAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	marker := "private-member@example.test"
	result := &model.ApproveResult{ResCode: "0000", CNO: "transaction42", Amount: "10000", AuthNo: "approval42", TranDate: "20260907", PayType: "11", CardNo: marker, ResMsg: marker, IssuerName: marker, AcquirerName: marker}
	logger.Log("order42", "approve_success", result, errors.New(marker))
	logger.Log("order42", "approve_fail", map[string]string{"reason": marker}, errors.New(marker))
	if err = logger.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), marker) {
		t.Fatal("private PG data written")
	}
	for _, expected := range []string{"transaction42", "10000", "approval42", "order42", "operation_failed"} {
		if !strings.Contains(string(data), expected) {
			t.Fatal("reconciliation evidence lost", expected)
		}
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("new audit file not private", err)
	}
}
