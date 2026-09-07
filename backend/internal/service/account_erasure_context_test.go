// account_erasure_context_test.go — Reject tampering, wrong keys and cross-request substitution.
package service

import (
	"bytes"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"testing"
)

func TestErasureContextAuthentication(t *testing.T) {
	c := testContextCipher(t)
	subject := model.ErasureExternalSubject{RequestID: 17, UserSeq: 42, Email: "synthetic@example.org"}
	encrypted, err := c.Seal(subject)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte(subject.Email)) {
		t.Fatal("plaintext persisted")
	}
	w := model.ErasureWork{RequestID: 17, UserSeq: 42}
	got, err := c.Open(w, encrypted)
	if err != nil || got.Email != subject.Email {
		t.Fatal("round trip failed", err)
	}
	if _, err = c.Open(model.ErasureWork{RequestID: 18, UserSeq: 42}, encrypted); err == nil {
		t.Fatal("cross-request substitution accepted")
	}
	if _, err = c.Open(model.ErasureWork{RequestID: 17, UserSeq: 43}, encrypted); err == nil {
		t.Fatal("cross-account substitution accepted")
	}
	wrong, _ := NewErasureContextCipher(strings.Repeat("cd", 32))
	if _, err = wrong.Open(w, encrypted); err == nil {
		t.Fatal("wrong key accepted")
	}
	encrypted[len(encrypted)-1] ^= 1
	if _, err = c.Open(w, encrypted); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err = c.Open(w, []byte("short")); err == nil {
		t.Fatal("truncated ciphertext accepted")
	}
}

type changingErasureContextStore struct {
	erasureStoreFake
	subject   model.ErasureExternalSubject
	refreshes int
}

func (f *changingErasureContextStore) ErasureExternalSubject(model.ErasureWork) (model.ErasureExternalSubject, error) {
	return f.subject, nil
}
func (f *changingErasureContextStore) RefreshErasureContext(id int64, data []byte) error {
	f.refreshes++
	return f.SaveErasureContext(id, data)
}
func TestPreDeletionContextRefreshBindsReviewedRetention(t *testing.T) {
	cipher := testContextCipher(t)
	old := model.ErasureExternalSubject{RequestID: 17, UserSeq: 42, Retentions: []model.DonationRetentionDecision{{OrderID: 1, SourceFingerprint: "old"}}}
	data, err := cipher.Seal(old)
	if err != nil {
		t.Fatal(err)
	}
	current := old
	current.Retentions = []model.DonationRetentionDecision{{OrderID: 1, SourceFingerprint: "new"}}
	store := &changingErasureContextStore{erasureStoreFake: erasureStoreFake{encrypted: data}, subject: current}
	svc := &AutomaticErasureService{Store: store, ContextCipher: cipher}
	work := model.ErasureWork{RequestID: 17, UserSeq: 42}
	if err = svc.preserveContext(&work); err != nil {
		t.Fatal(err)
	}
	opened, err := cipher.Open(work, store.encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if store.refreshes != 1 || opened.Retentions[0].SourceFingerprint != "new" || work.ContextRetentions[0].SourceFingerprint != "new" {
		t.Fatal("handoff and final database snapshot diverged")
	}
}
