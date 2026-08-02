package observability

import (
	"strings"
	"testing"
)

func TestRedactAuthJSONRemovesCredentialsRecursively(t *testing.T) {
	raw := []byte(`{
		"accessToken":"access-secret",
		"nested":{"refresh_token":"refresh-secret","identityToken":"identity-secret"},
		"authorizationCode":"code-secret",
		"safe":"visible"
	}`)
	redacted := string(RedactAuthJSON(raw))
	for _, secret := range []string{"access-secret", "refresh-secret", "identity-secret", "code-secret"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted output leaked %q: %s", secret, redacted)
		}
	}
	if !strings.Contains(redacted, `"safe":"visible"`) {
		t.Fatalf("safe value should remain visible: %s", redacted)
	}
}

func TestRedactAuthJSONDoesNotEchoMalformedInput(t *testing.T) {
	raw := []byte(`{"accessToken":"secret"`)
	redacted := string(RedactAuthJSON(raw))
	if strings.Contains(redacted, "secret") {
		t.Fatalf("malformed sensitive input was echoed: %s", redacted)
	}
}
