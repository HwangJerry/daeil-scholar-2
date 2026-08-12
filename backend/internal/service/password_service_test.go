// password_service_test.go — Unit tests for MySQL native password hashing
package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

func TestMysqlNativePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{
			name:     "known hash for test",
			password: "test",
			want:     "*94BDCEBE19083CE2A1F959FD02F964C7AF4CFC29",
		},
		{
			name:     "known hash for empty string",
			password: "",
			want:     "*BE1BDEC0AA74B4DCB079943E70528096CCA985F8",
		},
		{
			name:     "known hash for password",
			password: "password",
			want:     "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MysqlNativePassword(tt.password)
			if got != tt.want {
				t.Errorf("MysqlNativePassword(%q) = %q, want %q", tt.password, got, tt.want)
			}
		})
	}
}

func TestMysqlNativePasswordFormat(t *testing.T) {
	hash := MysqlNativePassword("anything")

	if !strings.HasPrefix(hash, "*") {
		t.Errorf("hash should start with *, got %q", hash)
	}
	if len(hash) != 41 {
		t.Errorf("hash length should be 41 (1 asterisk + 40 hex), got %d", len(hash))
	}
	if hash != strings.ToUpper(hash) {
		t.Errorf("hash should be uppercase, got %q", hash)
	}
}

func TestPasswordHasherArgon2idRoundTrip(t *testing.T) {
	hasher := newPasswordHasher(defaultArgon2idParameters, bytes.NewReader(make([]byte, argon2idSaltLength)))

	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("Hash() = %q, want Argon2id PHC format", hash)
	}

	verification, err := hasher.Verify("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !verification.Valid || verification.NeedsRehash || verification.UsedLegacy {
		t.Fatalf("Verify() = %+v, want valid canonical hash", verification)
	}

	verification, err = hasher.Verify("wrong password", hash)
	if err != nil {
		t.Fatalf("Verify(wrong password) error = %v", err)
	}
	if verification.Valid {
		t.Fatal("Verify(wrong password) unexpectedly succeeded")
	}
}

func TestPasswordHasherVerifiesLegacyAndRequiresRehash(t *testing.T) {
	verification, err := NewPasswordHasher().Verify("legacy password", MysqlNativePassword("legacy password"))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !verification.Valid || !verification.NeedsRehash || !verification.UsedLegacy {
		t.Fatalf("Verify() = %+v, want valid legacy hash requiring rehash", verification)
	}
}

func TestPasswordHasherVerifiesAlgorithmTaggedLegacyCredential(t *testing.T) {
	verification, err := NewPasswordHasher().VerifyCredential(
		model.PasswordAlgorithmMysqlNativePassword,
		"legacy password",
		MysqlNativePassword("legacy password"),
	)
	if err != nil {
		t.Fatalf("VerifyCredential() error = %v", err)
	}
	if !verification.Valid || !verification.NeedsRehash || !verification.UsedLegacy {
		t.Fatalf("VerifyCredential() = %+v", verification)
	}
}

func TestPasswordHasherRejectsUnknownCredentialAlgorithm(t *testing.T) {
	_, err := NewPasswordHasher().VerifyCredential(model.PasswordAlgorithm("UNKNOWN"), "password", "hash")
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("VerifyCredential() error = %v, want ErrInvalidPasswordHash", err)
	}
}

func TestPasswordHasherNeedsRehashWhenParametersDiffer(t *testing.T) {
	oldParameters := defaultArgon2idParameters
	oldParameters.MemoryKiB /= 2
	oldHasher := newPasswordHasher(oldParameters, bytes.NewReader(make([]byte, argon2idSaltLength)))
	hash, err := oldHasher.Hash("parameter migration password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	verification, err := NewPasswordHasher().Verify("parameter migration password", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !verification.Valid || !verification.NeedsRehash || verification.UsedLegacy {
		t.Fatalf("Verify() = %+v, want valid Argon2id hash requiring rehash", verification)
	}
}

func TestPasswordHasherNeedsRehashWhenSaltLengthDiffers(t *testing.T) {
	salt := make([]byte, 8)
	key := deriveArgon2idKey("salt migration password", salt, defaultArgon2idParameters)
	hash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		defaultArgon2idParameters.MemoryKiB,
		defaultArgon2idParameters.Iterations,
		defaultArgon2idParameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)

	verification, err := NewPasswordHasher().Verify("salt migration password", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !verification.Valid || !verification.NeedsRehash {
		t.Fatalf("Verify() = %+v, want rehash for noncanonical salt length", verification)
	}
}

func TestPasswordHasherRejectsMalformedHash(t *testing.T) {
	_, err := NewPasswordHasher().Verify("password", "$argon2id$v=19$m=broken$not-a-hash")
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("Verify() error = %v, want ErrInvalidPasswordHash", err)
	}
}

func TestPasswordHasherRejectsHashAboveMemoryEnvelope(t *testing.T) {
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, argon2idSaltLength))
	key := base64.RawStdEncoding.EncodeToString(make([]byte, argon2idKeyLength))
	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=2,p=1$%s$%s", argon2idVersion, maxArgon2idMemoryKiB+1, salt, key)

	_, err := NewPasswordHasher().Verify("submitted password", hash)
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("Verify() error = %v, want ErrInvalidPasswordHash", err)
	}
}

func TestPasswordHasherUsesGlobalSemaphore(t *testing.T) {
	if cap(passwordHashSemaphore) != 1 {
		t.Fatalf("password hash semaphore capacity = %d, want 1", cap(passwordHashSemaphore))
	}
	for range cap(passwordHashSemaphore) {
		passwordHashSemaphore <- struct{}{}
	}
	t.Cleanup(func() {
		for len(passwordHashSemaphore) > 0 {
			<-passwordHashSemaphore
		}
	})

	result := make(chan error, 1)
	go func() {
		_, err := newPasswordHasher(defaultArgon2idParameters, bytes.NewReader(make([]byte, argon2idSaltLength))).Hash("queued password")
		result <- err
	}()

	select {
	case err := <-result:
		t.Fatalf("Hash() bypassed global semaphore: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	<-passwordHashSemaphore
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Hash() error after semaphore release = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Hash() did not resume after semaphore release")
	}
}

func BenchmarkPasswordHasherOneCoreEnvelope(b *testing.B) {
	hasher := NewPasswordHasher()
	for b.Loop() {
		if _, err := hasher.Hash("benchmark password"); err != nil {
			b.Fatal(err)
		}
	}
}
