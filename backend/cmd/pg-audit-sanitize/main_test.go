// main_test.go — Preserve source/evidence and publish no partial or overwritten previews.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewMinimizesWithoutChangingSourceOrExistingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.log")
	output := filepath.Join(root, "review.log")
	raw := `{"ts":"2026-09-07T12:00:00+09:00","order_no":"order42","event":"approve_success","data":{"CNO":"transaction42","Amount":"10000","CardNo":"private@example.test","ResMsg":"private@example.test"},"error":"private@example.test"}` + "\n"
	if err := os.WriteFile(input, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	count, err := run(input, output)
	if err != nil || count != 1 {
		t.Fatal(count, err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private@example.test") || !strings.Contains(string(data), "transaction42") || !strings.Contains(string(data), "10000") {
		t.Fatal("privacy or evidence failure")
	}
	source, _ := os.ReadFile(input)
	if string(source) != raw {
		t.Fatal("source modified")
	}
	info, _ := os.Stat(output)
	if info.Mode().Perm() != 0600 {
		t.Fatal("preview not private")
	}
	if _, err = run(input, output); err == nil {
		t.Fatal("existing output overwritten")
	}
	after, _ := os.ReadFile(output)
	if string(after) != string(data) {
		t.Fatal("existing output changed")
	}
	partial := filepath.Join(root, "partial.log")
	if err = os.WriteFile(input, []byte(raw+`{"unknown":"financial evidence"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = run(input, partial); err == nil {
		t.Fatal("unknown record silently discarded")
	}
	if _, err = os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("partial result published")
	}
}
