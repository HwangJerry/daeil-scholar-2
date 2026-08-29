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
