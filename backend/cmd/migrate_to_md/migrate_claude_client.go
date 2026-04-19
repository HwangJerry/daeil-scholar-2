// migrate_claude_cli — invokes the claude CLI subprocess for HTML→Markdown conversion
package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// callClaude calls the local `claude` CLI with systemPrompt prepended to the user content.
// It returns the trimmed text response, or an error if the subprocess fails.
func callClaude(userContent string) (string, error) {
	fullPrompt := systemPrompt + "\n\n" + userContent
	cmd := exec.Command("claude", "-p", fullPrompt)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude CLI failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}
