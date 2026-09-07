// account_erasure_test.go — Boundary tests for erasure storage and statutory clocks.
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDonationRetentionUsesRecordedClockNotWithdrawalDate(t *testing.T) {
	start := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(10, 0, 0)
	decision := model.DonationRetentionDecision{OrderID: 10, Basis: "ledger_10y", BasisDate: &start, Until: &end, Evidence: "accountant-review-10"}
	if err := ValidateDonationRetention(decision); err != nil {
		t.Fatal(err)
	}
	wrong := time.Now().AddDate(10, 0, 0)
	decision.Until = &wrong
	if ValidateDonationRetention(decision) == nil {
		t.Fatal("withdrawal based retention accepted")
	}
	decision.Basis = "receipt_5y"
	end = start.AddDate(5, 0, 0)
	decision.Until = &end
	if err := ValidateDonationRetention(decision); err != nil {
		t.Fatal(err)
	}
	decision.Basis = "erase"
	if ValidateDonationRetention(decision) == nil {
		t.Fatal("erase decision retained data")
	}
	decision.BasisDate = nil
	decision.Until = nil
	if err := ValidateDonationRetention(decision); err != nil {
		t.Fatal(err)
	}
	decision.Evidence = ""
	if ValidateDonationRetention(decision) == nil {
		t.Fatal("unreviewed decision accepted")
	}
}

func TestDonationArchiveAuthenticatedEncryption(t *testing.T) {
	key := strings.Repeat("01", 32)
	seal, err := DonationArchiveSealer(key)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"donorName":"Synthetic Donor","amount":100}`)
	first, err := seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := seal(plain)
	if string(first) == string(second) || strings.Contains(string(first), "Synthetic Donor") {
		t.Fatal("archive exposed data or reused nonce")
	}
	block, _ := aes.NewCipher([]byte(strings.Repeat("\x01", 32)))
	aead, _ := cipher.NewGCM(block)
	result, err := aead.Open(nil, first[:aead.NonceSize()], first[aead.NonceSize():], []byte("DFLH_DONATION_LEGAL_ARCHIVE_V1"))
	if err != nil || string(result) != string(plain) {
		t.Fatal("archive not recoverable with authorized key")
	}
	first[len(first)-1] ^= 1
	if _, err = aead.Open(nil, first[:aead.NonceSize()], first[aead.NonceSize():], []byte("DFLH_DONATION_LEGAL_ARCHIVE_V1")); err == nil {
		t.Fatal("tampering accepted")
	}
}

func TestErasureFilesRejectTraversalSymlinkAndForeignOrigins(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "keep"), []byte("other user"), 0600)
	os.Mkdir(filepath.Join(root, "profile"), 0700)
	own := filepath.Join(root, "profile", "own.jpg")
	os.WriteFile(own, []byte("test"), 0600)
	store := &AccountErasureFiles{UploadRoot: root, SiteOrigin: "https://app.example.org"}
	if err := store.EraseURL("/uploads/profile/own.jpg"); err != nil {
		t.Fatal(err)
	}
	if err := store.EraseURL("/uploads/profile/own.jpg"); err != nil {
		t.Fatal("retry failed", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/uploads/../keep", "/uploads/%2e%2e/keep", "/uploads/link/keep", "https://foreign.example/uploads/profile/own.jpg", "/uploads/profile"} {
		if store.EraseURL(url) == nil {
			t.Fatalf("unsafe path accepted: %s", url)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "keep")); err != nil {
		t.Fatal("unrelated file erased")
	}
}

func TestLedgerRetentionTemplateRequiresConfirmedPolicy(t *testing.T) {
	if LedgerRetentionTemplate(false, true, 12, "review") != nil {
		t.Fatal("unconfirmed legal applicability enabled")
	}
	if LedgerRetentionTemplate(true, false, 12, "review") != nil {
		t.Fatal("unverified receipt originals enabled")
	}
	template := LedgerRetentionTemplate(true, true, 2, "verified fiscal year review")
	date := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
	decision, err := template(1, date)
	if err != nil || decision.BasisDate.Format("2006-01-02") != "2024-02-29" || decision.Until.Format("2006-01-02") != "2034-02-28" {
		t.Fatal("fiscal calendar wrong", decision, err)
	}
	if err = ValidateDonationRetention(decision); err != nil {
		t.Fatal(err)
	}
}
