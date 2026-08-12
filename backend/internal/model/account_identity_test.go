package model

import (
	"errors"
	"testing"
)

func TestNewEmailIdentityNormalizesVerifiedEmail(t *testing.T) {
	identity, err := NewIdentity(42, IdentityProviderEmail, " Member@Example.COM ", " Member@Example.COM ", true)
	if err != nil {
		t.Fatal(err)
	}
	if identity.SubjectKey != "member@example.com" || identity.NormalizedEmail == nil || *identity.NormalizedEmail != "member@example.com" {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestNewIdentityRejectsEmailMetadataForNonEmailProvider(t *testing.T) {
	_, err := NewIdentity(42, IdentityProviderKakao, "provider-subject", "provider@example.com", true)
	if !errors.Is(err, ErrNormalizedEmailProvider) {
		t.Fatalf("error = %v, want %v", err, ErrNormalizedEmailProvider)
	}
}

func TestNewIdentityRejectsUnverifiedEmailAuthority(t *testing.T) {
	_, err := NewIdentity(42, IdentityProviderEmail, "member@example.com", "member@example.com", false)
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("error = %v, want %v", err, ErrEmailNotVerified)
	}
}

func TestIdentityProviderPasswordAndSocialBoundaries(t *testing.T) {
	if !IdentityProviderEmail.SupportsPassword() || !IdentityProviderLocalUsername.SupportsPassword() {
		t.Fatal("EMAIL and LOCAL_USERNAME must support password credentials")
	}
	if IdentityProviderKakao.SupportsPassword() || IdentityProviderApple.SupportsPassword() {
		t.Fatal("social providers must not support password credentials")
	}
	if !IdentityProviderKakao.SupportsProviderCredential() || !IdentityProviderApple.SupportsProviderCredential() {
		t.Fatal("KAKAO and APPLE must support provider credentials")
	}
	if IdentityProviderEmail.SupportsProviderCredential() || IdentityProviderLocalUsername.SupportsProviderCredential() {
		t.Fatal("password providers must not support provider credentials")
	}
}
