package golden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertUpdateAndCompare(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GOLDEN_UPDATE", "1")
	Assert(t, "login", []byte(`{"token":"first","id":42}`))
	t.Setenv("GOLDEN_UPDATE", "")
	Assert(t, "login", []byte(`{"id":42,"token":"second"}`))
	t.Setenv("GOLDEN_UPDATE", "1")
	Assert(t, "login", []byte(`{"id":43}`))
	t.Setenv("GOLDEN_UPDATE", "")
	Assert(t, "login", []byte(`{"id":43}`))
}

func TestCompareRequiresExplicitUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testdata", "golden", "response.json")
	if err := compare(path, []byte("{}\n"), false); err == nil || !strings.Contains(err.Error(), "GOLDEN_UPDATE=1") {
		t.Fatalf("missing fixture error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("comparison created missing fixture")
	}
	if err := compare(path, []byte("{}\n"), true); err != nil {
		t.Fatal(err)
	}
	if err := compare(path, []byte("null\n"), false); err == nil || !strings.Contains(err.Error(), "golden mismatch") {
		t.Fatalf("mismatch error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{}\n" {
		t.Fatal("comparison rewrote mismatched fixture")
	}
}
