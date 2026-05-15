// history.go — Domain model for scholarship history entries
package model

// HistoryEntry represents a row in the HISTORY_ENTRY table.
type HistoryEntry struct {
	HESeq        int    `db:"HE_SEQ"         json:"heSeq"`
	HEEventDate  string `db:"HE_EVENT_DATE"  json:"eventDate"` // "YYYY-MM-DD"
	HEText       string `db:"HE_TEXT"        json:"text"`
	HESortOrder  int    `db:"HE_SORT_ORDER"  json:"sortOrder"`
}

// HistoryYearGroup groups entries by year for the public API response.
type HistoryYearGroup struct {
	Year  int            `json:"year"`
	Items []HistoryEntry `json:"items"`
}

// HistoryUpsertRequest is the request body for create / update.
type HistoryUpsertRequest struct {
	EventDate string `json:"eventDate"` // "YYYY-MM-DD"
	Text      string `json:"text"`
	SortOrder int    `json:"sortOrder"`
}
