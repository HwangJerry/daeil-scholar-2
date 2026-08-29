package handler

import "time"

// parseUTCISOtoDB converts an RFC3339 timestamp to the database DATETIME format.
func parseUTCISOtoDB(iso string) (string, error) {
	parsed, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format("2006-01-02 15:04:05"), nil
}
