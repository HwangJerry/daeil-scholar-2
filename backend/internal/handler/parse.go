package handler

import (
	"strconv"
	"strings"
)

func parseCursor(value string) int {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "seq_") {
		value = strings.TrimPrefix(value, "seq_")
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseIntParam(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

// parseStrictSeqCursor reads the feed-style "seq_<n>" cursor (a bare positive
// number is accepted too, as parseCursor does). An empty value means the first
// page; anything else that is not a positive SEQ is rejected instead of being
// silently treated as the first page.
func parseStrictSeqCursor(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, true
	}
	parsed, err := strconv.Atoi(strings.TrimPrefix(value, "seq_"))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}
