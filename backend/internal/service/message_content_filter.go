// message_content_filter.go — Pre-publication filtering for direct-message text.
package service

import (
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"
)

// This initial rule list supplements reporting and human review; it is not a
// complete language classifier. Extend through MESSAGE_BLOCKED_PHRASES (comma-separated).
var defaultBlockedMessagePhrases = []string{"씨발", "개새끼", "죽여버릴", "kill yourself"}

type MessageContentFilter struct{ phrases []string }

func NewMessageContentFilter(additional []string) MessageContentFilter {
	filter := MessageContentFilter{}
	for _, phrase := range append(append([]string{}, defaultBlockedMessagePhrases...), additional...) {
		if normalized := normalizeMessageForFiltering(phrase); normalized != "" {
			filter.phrases = append(filter.phrases, normalized)
		}
	}
	return filter
}

func normalizeMessageForFiltering(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, norm.NFKC.String(text))
}

func (filter MessageContentFilter) Allows(content string) bool {
	text := normalizeMessageForFiltering(content)
	for _, phrase := range filter.phrases {
		if strings.Contains(text, phrase) {
			return false
		}
	}
	return true
}

func (s *MessageService) ConfigureContentFilter(additional []string) {
	s.contentFilter = NewMessageContentFilter(additional)
}
