package handler

import (
	"strings"
	"testing"
)

func TestNormalizeCanonicalMobileSocialLinkRequestUsesEmailCredentials(t *testing.T) {
	request := socialLinkRequest{
		LinkToken: " fixture-link-token ",
		Email:     " member@example.com ",
		Password:  "fixture-password",
	}

	canonical := normalizeCanonicalMobileSocialLinkRequest(&request)

	if !canonical {
		t.Fatal("canonical mobile request was not detected")
	}
	if request.Token != "fixture-link-token" {
		t.Fatalf("token = %q", request.Token)
	}
	if request.Mode != "merge" || request.Client != "mobile" {
		t.Fatalf("mode/client = %q/%q", request.Mode, request.Client)
	}
	if request.ExistingEmail != "member@example.com" {
		t.Fatalf("existingEmail = %q", request.ExistingEmail)
	}
	if request.ExistingPassword != "fixture-password" {
		t.Fatal("existing password was not forwarded")
	}
	if request.ExistingUSRID != "" {
		t.Fatalf("legacy existingUsrId must remain empty: %q", request.ExistingUSRID)
	}
}

func TestNormalizeCanonicalMobileSocialLinkRequestRejectsPartialShape(t *testing.T) {
	for _, body := range []socialLinkRequest{
		{LinkToken: "token", Email: "member@example.com"},
		{LinkToken: "token", Password: "password"},
		{Email: "member@example.com", Password: "password"},
	} {
		request := body
		if normalizeCanonicalMobileSocialLinkRequest(&request) {
			t.Fatalf("partial shape detected as canonical: %#v", request)
		}
	}
}

func TestCanonicalMobileSocialLinkWireKeysRemainClosed(t *testing.T) {
	body := `{"linkToken":"fixture","email":"member@example.com","password":"secret"}`
	for _, legacyKey := range []string{"existingUsrId", "existingPassword", "mode", "client", "phone", "fn", "fmDept"} {
		if strings.Contains(body, legacyKey) {
			t.Fatalf("canonical body includes legacy key %q", legacyKey)
		}
	}
}
