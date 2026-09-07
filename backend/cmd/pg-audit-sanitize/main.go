// pg-audit-sanitize — Create a private reviewed copy; never rewrite a live audit log.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/dflh-saf/backend/internal/service"
	"io"
	"os"
	"path/filepath"
)

const maxAuditLineBytes = 1 << 20
const maxAuditFileBytes = 50 << 20

func main() {
	input := flag.String("input", "", "closed or stable PG JSON-lines source")
	output := flag.String("output", "", "new private preview path; must not already exist")
	flag.Parse()
	count, err := run(*input, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "PG audit preview failed; source and existing output were not replaced.")
		os.Exit(1)
	}
	fmt.Printf("Prepared %d minimized records; source unchanged. Live replacement still requires a controlled writer stop.\n", count)
}

func run(input, output string) (int, error) {
	source, err := os.Open(input)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	before, err := source.Stat()
	if err != nil {
		return 0, err
	}
	if !before.Mode().IsRegular() || before.Size() > maxAuditFileBytes {
		return 0, fmt.Errorf("invalid input file")
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".pg-audit-review-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	scanner := bufio.NewScanner(io.LimitReader(source, maxAuditFileBytes+1))
	scanner.Buffer(make([]byte, 65536), maxAuditLineBytes)
	count := 0
	totalBytes := 0
	for scanner.Scan() {
		totalBytes += len(scanner.Bytes()) + 1
		if totalBytes > maxAuditFileBytes {
			return 0, fmt.Errorf("source exceeds limit")
		}
		if len(scanner.Bytes()) == 0 {
			continue
		}
		record, err := service.SanitizePGAuditRecord(scanner.Bytes())
		if err != nil {
			return 0, err
		}
		if _, err = tmp.Write(append(record, '\n')); err != nil {
			return 0, err
		}
		count++
	}
	if err = scanner.Err(); err != nil {
		return 0, err
	}
	after, err := source.Stat()
	if err != nil {
		return 0, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return 0, fmt.Errorf("source changed during review")
	}
	if err = tmp.Sync(); err != nil {
		return 0, err
	}
	if err = tmp.Close(); err != nil {
		return 0, err
	}
	// Hard-link is atomic and refuses an existing output, unlike Rename.
	if err = os.Link(tmp.Name(), output); err != nil {
		return 0, err
	}
	return count, nil
}
