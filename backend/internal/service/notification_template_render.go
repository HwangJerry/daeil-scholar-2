// notification_template_render.go — Placeholder scanning and substitution for notification texts.
package service

import (
	"regexp"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	// templateBraceTokenPattern finds every `{...}` token, including malformed
	// ones, so a typo such as `{cod}` is rejected instead of being shipped to a
	// member as literal text.
	templateBraceTokenPattern = regexp.MustCompile(`\{[^{}]*\}`)
	// templatePlaceholderNamePattern is the accepted placeholder spelling.
	templatePlaceholderNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
)

// templateBraceTokens returns every `{...}` token in the text, braces included.
func templateBraceTokens(text string) []string {
	return templateBraceTokenPattern.FindAllString(text, -1)
}

// templateTokenName strips the braces from a token found by templateBraceTokens.
func templateTokenName(token string) string {
	return strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
}

// templateContainsPlaceholder reports whether the text uses `{name}`.
func templateContainsPlaceholder(text, name string) bool {
	return strings.Contains(text, "{"+name+"}")
}

// renderTemplateText substitutes every placeholder the definition allows.
//
// A placeholder absent from vars renders as an empty string: a missing sender
// name must never leak the literal `{senderName}` into a notification. Tokens
// that are not allowed for this key are left untouched, because validation
// rejects them before they can be stored.
func renderTemplateText(text string, definition model.NotificationTemplateDefinition, vars map[string]string) string {
	if len(definition.Placeholders) == 0 {
		return text
	}
	replacements := make([]string, 0, len(definition.Placeholders)*2)
	for _, placeholder := range definition.Placeholders {
		replacements = append(replacements, "{"+placeholder.Name+"}", vars[placeholder.Name])
	}
	return strings.NewReplacer(replacements...).Replace(text)
}

// renderTemplateWithSamples substitutes each placeholder with its worst-case
// sample value, which is what the length rules must be measured against.
func renderTemplateWithSamples(text string, definition model.NotificationTemplateDefinition) string {
	samples := make(map[string]string, len(definition.Placeholders))
	for _, placeholder := range definition.Placeholders {
		samples[placeholder.Name] = placeholder.Sample
	}
	return renderTemplateText(text, definition, samples)
}
