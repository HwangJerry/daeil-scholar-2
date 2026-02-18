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

func parseExcludeAds(value string) []int {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			continue
		}
		result = append(result, parsed)
	}
	return result
}

func parseIntParam(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}
