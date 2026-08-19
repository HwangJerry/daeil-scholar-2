// mvp_contract_fixture_test.go — Validates canonical cross-platform MVP API fixtures.
package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

var canonicalFixtureNames = []string{
	"alumni-search.json",
	"auth-authenticated.json",
	"auth-unsubmitted.json",
	"auth-link-required.json",
	"auth-pending.json",
	"auth-rejected.json",
	"donation-import-validation-error.json",
	"donation-summary.json",
	"error-approval-required.json",
	"message-event.json",
	"message-send.json",
	"push-device.json",
	"push-message.json",
	"push-preferences.json",
}

func TestCanonicalFixtureSetIsCompleteAndValidJSON(t *testing.T) {
	entries, err := os.ReadDir(canonicalFixtureDir())
	if err != nil {
		t.Fatalf("read canonical fixture directory: %v", err)
	}

	actual := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			actual = append(actual, entry.Name())
		}
	}
	sort.Strings(actual)

	expected := append([]string(nil), canonicalFixtureNames...)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("fixture set mismatch\nactual:   %v\nexpected: %v", actual, expected)
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			fixture := readFixture(t, name)
			if len(fixture) == 0 {
				t.Fatal("fixture must be a non-empty JSON object")
			}
		})
	}
}

func TestRestrictedAuthFixturesContainSessionAndVerification(t *testing.T) {
	for name, status := range map[string]string{
		"auth-unsubmitted.json": "unsubmitted",
		"auth-pending.json":     "pending",
		"auth-rejected.json":    "rejected",
	} {
		t.Run(status, func(t *testing.T) {
			fixture := readFixture(t, name)
			if fixture["status"] != "authenticated" {
				t.Fatalf("auth status = %v, want authenticated", fixture["status"])
			}
			session := objectField(t, fixture, "session")
			user := objectField(t, session, "user")
			verification := objectField(t, user, "verification")
			if verification["status"] != status {
				t.Fatalf("verification status = %v, want %s", verification["status"], status)
			}
			if session["accessToken"] == nil || session["refreshToken"] == nil {
				t.Fatal("restricted auth fixture must include a limited session")
			}
		})
	}
}

func TestPublicDonationSummaryContainsRestoredSnapshotContract(t *testing.T) {
	fixture := readFixture(t, "donation-summary.json")
	wantKeys := []string{"achievementRate", "displayAmount", "donorCount", "goalAmount", "snapshotDate", "tierThresholds"}
	if !reflect.DeepEqual(sortedKeys(fixture), wantKeys) {
		t.Fatalf("public donation keys = %v, want %v", sortedKeys(fixture), wantKeys)
	}
	tierThresholds := objectField(t, fixture, "tierThresholds")
	wantTierKeys := []string{"blooming", "fruiting", "sapling", "sprout", "tree"}
	if !reflect.DeepEqual(sortedKeys(tierThresholds), wantTierKeys) {
		t.Fatalf("tier threshold keys = %v, want %v", sortedKeys(tierThresholds), wantTierKeys)
	}
}

func TestBlockedSendFixtureDoesNotRevealDeliveryState(t *testing.T) {
	fixture := readFixture(t, "message-send.json")
	if fixture["status"] != "accepted" {
		t.Fatalf("message status = %v, want accepted", fixture["status"])
	}
	forbidden := []string{"blocked", "delivered", "recipientBlocked", "suppressed"}
	for _, key := range forbidden {
		if _, exists := fixture[key]; exists {
			t.Fatalf("message send fixture leaks delivery state through %q", key)
		}
	}
}

func canonicalFixtureDir() string {
	return filepath.Join("..", "..", "..", "docs", "contracts", "fixtures")
}

func readFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(canonicalFixtureDir(), name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return fixture
}

func objectField(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("field %s must be an object", key)
	}
	return value
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
