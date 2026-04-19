// markdown_normalizer — normalizes CONTENTS_MD list formatting for correct goldmark rendering
package migrator

import (
	"regexp"
	"strings"
)

var (
	numberedRe   = regexp.MustCompile(`^\d+\. `)
	bulletStarRe = regexp.MustCompile(`^\* `)
	bulletDashRe = regexp.MustCompile(`^- `)
	colonRe      = regexp.MustCompile(`^: `)
	headingRe    = regexp.MustCompile(`^## `)
)

// NormalizeMarkdownLists fixes indentation of bullet/colon lines that appear inside
// numbered-list sections. Without normalization, goldmark treats those lines as
// separate top-level blocks, which breaks <ol start> continuity.
//
// Rules (only applied when inside a numbered section):
//   - "* text" → "   - text"  (star bullet becomes indented dash)
//   - "- text" → "   - text"  (dash+space bullet becomes indented dash)
//   - ": text" → appended to the previous pending bullet line
//
// A numbered section starts on a "N. " line and ends on a "## " heading.
// Empty lines preserve state (numbered section stays active across blank lines).
func NormalizeMarkdownLists(md string) string {
	lines := strings.Split(md, "\n")
	out := make([]string, 0, len(lines))

	inNumberedSection := false
	pendingLine := ""

	flushPending := func() {
		if pendingLine != "" {
			out = append(out, pendingLine)
			pendingLine = ""
		}
	}

	for _, line := range lines {
		switch {
		case numberedRe.MatchString(line):
			flushPending()
			inNumberedSection = true
			out = append(out, line)
		case headingRe.MatchString(line):
			flushPending()
			inNumberedSection = false
			out = append(out, line)
		case line == "":
			flushPending()
			out = append(out, line)
		case inNumberedSection && bulletStarRe.MatchString(line):
			flushPending()
			pendingLine = "   - " + line[2:]
		case inNumberedSection && bulletDashRe.MatchString(line):
			flushPending()
			pendingLine = "   - " + line[2:]
		case inNumberedSection && colonRe.MatchString(line):
			if pendingLine != "" {
				pendingLine += " " + line
			} else {
				out = append(out, line)
			}
		default:
			flushPending()
			out = append(out, line)
		}
	}
	flushPending()

	return strings.Join(out, "\n")
}
