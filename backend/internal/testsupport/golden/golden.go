package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var fixtureName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Assert compares a normalized response with testdata/golden/<name>.json in the
// calling package. GOLDEN_UPDATE=1 creates or rewrites the fixture explicitly.
func Assert(t testing.TB, name string, body []byte, volatileKeys ...string) {
	t.Helper()
	if !fixtureName.MatchString(name) {
		t.Fatalf("invalid golden name %q: use letters, digits, hyphens or underscores", name)
	}
	normalized, err := Normalize(body, volatileKeys...)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "golden", name+".json")
	if err := compare(path, normalized, os.Getenv("GOLDEN_UPDATE") == "1"); err != nil {
		t.Fatal(err)
	}
}

func compare(path string, actual []byte, update bool) error {
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, actual, 0o644)
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read golden %s (use GOLDEN_UPDATE=1 to create it): %w", path, err)
	}
	if !bytes.Equal(expected, actual) {
		return fmt.Errorf("golden mismatch: %s (use GOLDEN_UPDATE=1 to update)\nwant:\n%s\ngot:\n%s", path, expected, actual)
	}
	return nil
}
