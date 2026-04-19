// html_to_markdown — converts legacy HTML post content to clean Markdown text
package converter

import (
	"html"
	"regexp"
	"strings"
)

var (
	styleRe    = regexp.MustCompile(`(?si)<style[^>]*>.*?</style>`)
	scriptRe   = regexp.MustCompile(`(?si)<script[^>]*>.*?</script>`)
	h1Re       = regexp.MustCompile(`(?si)<h1[^>]*>(.*?)</h1>`)
	h2Re       = regexp.MustCompile(`(?si)<h2[^>]*>(.*?)</h2>`)
	h3Re       = regexp.MustCompile(`(?si)<h3[^>]*>(.*?)</h3>`)
	h4Re       = regexp.MustCompile(`(?si)<h4[^>]*>(.*?)</h4>`)
	boldRe     = regexp.MustCompile(`(?si)<(?:strong|b)[^>]*>(.*?)</(?:strong|b)>`)
	italicRe   = regexp.MustCompile(`(?si)<(?:em|i)[^>]*>(.*?)</(?:em|i)>`)
	imgRe      = regexp.MustCompile(`(?si)<img[^>]+src="([^"]*)"[^>]*>`)
	aRe        = regexp.MustCompile(`(?si)<a[^>]+href="([^"]*)"[^>]*>(.*?)</a>`)
	brRe       = regexp.MustCompile(`(?si)<br\s*/?>`)
	hrRe       = regexp.MustCompile(`(?si)<hr[^>]*/?>`)
	pOpenRe    = regexp.MustCompile(`(?si)<p[^>]*>`)
	pCloseRe   = regexp.MustCompile(`</p>`)
	divOpenRe  = regexp.MustCompile(`(?si)<div[^>]*>`)
	divCloseRe = regexp.MustCompile(`</div>`)
	liRe       = regexp.MustCompile(`(?si)<li[^>]*>(.*?)</li>`)
	tagRe      = regexp.MustCompile(`<[^>]*>`)
	multiNLRe  = regexp.MustCompile(`\n{3,}`)
)

// FromLegacyPost converts a LEGACY HTML post (subject + contents) to clean Markdown.
// It handles both raw HTML and plain-text content gracefully.
func FromLegacyPost(subject, contents string) string {
	h := contents

	// Remove style/script blocks before anything else
	h = styleRe.ReplaceAllString(h, "")
	h = scriptRe.ReplaceAllString(h, "")

	// Structural headings
	h = h1Re.ReplaceAllString(h, "# $1\n\n")
	h = h2Re.ReplaceAllString(h, "## $1\n\n")
	h = h3Re.ReplaceAllString(h, "### $1\n\n")
	h = h4Re.ReplaceAllString(h, "#### $1\n\n")

	// Inline emphasis
	h = boldRe.ReplaceAllString(h, "**$1**")
	h = italicRe.ReplaceAllString(h, "*$1*")

	// Images before links (both use angle-bracket attrs)
	h = imgRe.ReplaceAllString(h, "![]($1)\n")

	// Hyperlinks
	h = aRe.ReplaceAllString(h, "[$2]($1)")

	// Block-level structure → newlines
	h = brRe.ReplaceAllString(h, "\n")
	h = hrRe.ReplaceAllString(h, "\n---\n")
	h = pOpenRe.ReplaceAllString(h, "")
	h = pCloseRe.ReplaceAllString(h, "\n\n")
	h = divOpenRe.ReplaceAllString(h, "")
	h = divCloseRe.ReplaceAllString(h, "\n")

	// List items
	h = liRe.ReplaceAllString(h, "- $1\n")

	// Strip all remaining HTML tags
	h = tagRe.ReplaceAllString(h, "")

	// Decode HTML entities (e.g. &nbsp; &lt; &amp;)
	h = html.UnescapeString(h)

	// Normalize non-breaking spaces to regular spaces
	h = strings.ReplaceAll(h, "\u00a0", " ")

	// Collapse excess blank lines (max 1 blank line between paragraphs)
	h = multiNLRe.ReplaceAllString(h, "\n\n")
	h = strings.TrimSpace(h)

	// Prepend subject as H2 heading if content has no heading
	if !strings.HasPrefix(h, "#") {
		h = "## " + subject + "\n\n" + h
	}

	return h
}
