// mobile_app_event_repo.go — Persistence for mobile business-event analytics.
package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type MobileAppEventRepository struct {
	db *sqlx.DB
}

func NewMobileAppEventRepository(db *sqlx.DB) *MobileAppEventRepository {
	return &MobileAppEventRepository{db: db}
}

// InsertBatch stores all events in one transaction so a client retry never
// observes a partially accepted batch.
func (r *MobileAppEventRepository) InsertBatch(events []model.MobileAppEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statement, err := tx.Preparex(`
		INSERT INTO ALUMNI_MOBILE_APP_EVENT (
			PLATFORM, EVENT_TYPE, USER_ID, APP_VERSION, OS_VERSION,
			DEVICE_MODEL, OCCURRED_AT, CREATED_AT
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer statement.Close()

	for _, event := range events {
		if _, err := statement.Exec(
			event.Platform,
			event.EventType,
			event.UserID,
			event.AppVersion,
			event.OSVersion,
			event.DeviceModel,
			event.OccurredAt,
			event.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSummary returns counts grouped by platform and event type for a filtered
// half-open interval.
func (r *MobileAppEventRepository) GetSummary(filter model.MobileAppEventSummaryFilter) ([]model.MobileAppEventSummary, error) {
	query := `
		SELECT PLATFORM, EVENT_TYPE, COUNT(*) AS EVENT_COUNT
		FROM ALUMNI_MOBILE_APP_EVENT
		WHERE OCCURRED_AT >= ? AND OCCURRED_AT < ?`
	args := []interface{}{filter.From, filter.ToExclusive}
	if filter.Platform != "" {
		query += " AND PLATFORM = ?"
		args = append(args, filter.Platform)
	}
	if filter.EventType != "" {
		query += " AND EVENT_TYPE = ?"
		args = append(args, filter.EventType)
	}
	query += `
		GROUP BY PLATFORM, EVENT_TYPE
		ORDER BY EVENT_TYPE ASC, PLATFORM ASC`

	items := make([]model.MobileAppEventSummary, 0)
	err := r.db.Select(&items, strings.TrimSpace(query), args...)
	return items, err
}
