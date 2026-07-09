package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	PushOutboxStatusPending    = "PENDING"
	PushOutboxStatusProcessing = "PROCESSING"
	PushOutboxStatusSent       = "SENT"
	PushOutboxStatusFailed     = "FAILED"
	PushOutboxStatusDead       = "DEAD"
)

type PushOutboxRepository struct {
	DB *sqlx.DB
}

type PushOutboxInsert struct {
	EventType       string
	EventID         string
	UsrSeq          int
	MDTSeq          int
	DeviceToken     string
	APNsEnvironment string
	BundleID        string
	Title           string
	Body            string
	PayloadJSON     string
	NextAttemptAt   time.Time
}

type PushOutboxJob struct {
	POSeq            int            `db:"PO_SEQ"`
	EventType        string         `db:"EVENT_TYPE"`
	EventID          string         `db:"EVENT_ID"`
	UsrSeq           int            `db:"USR_SEQ"`
	MDTSeq           int            `db:"MDT_SEQ"`
	DeviceToken      string         `db:"DEVICE_TOKEN"`
	APNsEnvironment  string         `db:"APNS_ENVIRONMENT"`
	BundleID         sql.NullString `db:"BUNDLE_ID"`
	Title            string         `db:"TITLE"`
	Body             string         `db:"BODY"`
	PayloadJSON      string         `db:"PAYLOAD_JSON"`
	Status           string         `db:"STATUS"`
	AttemptCount     int            `db:"ATTEMPT_COUNT"`
	NextAttemptAt    time.Time      `db:"NEXT_ATTEMPT_AT"`
	LastErrorCode    sql.NullString `db:"LAST_ERROR_CODE"`
	LastErrorMessage sql.NullString `db:"LAST_ERROR_MESSAGE"`
	CreatedAt        time.Time      `db:"CREATED_AT"`
	UpdatedAt        time.Time      `db:"UPDATED_AT"`
	SentAt           sql.NullTime   `db:"SENT_AT"`
}

type PushOutboxStats struct {
	PendingCount      int `db:"PENDING_COUNT"`
	DeadCount         int `db:"DEAD_COUNT"`
	OldestPendingAge  time.Duration
	OldestPendingTime sql.NullTime `db:"OLDEST_PENDING_AT"`
}

func NewPushOutboxRepository(db *sqlx.DB) *PushOutboxRepository {
	return &PushOutboxRepository{DB: db}
}

func (r *PushOutboxRepository) Enqueue(ctx context.Context, job PushOutboxInsert) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO ALUMNI_PUSH_OUTBOX
			(EVENT_TYPE, EVENT_ID, USR_SEQ, MDT_SEQ, DEVICE_TOKEN, APNS_ENVIRONMENT, BUNDLE_ID,
			 TITLE, BODY, PAYLOAD_JSON, STATUS, ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', 0, NOW(), NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			PO_SEQ = PO_SEQ
	`, job.EventType, job.EventID, job.UsrSeq, job.MDTSeq, job.DeviceToken, job.APNsEnvironment,
		nullableOutboxString(job.BundleID), job.Title, job.Body, job.PayloadJSON)
	return err
}

func (r *PushOutboxRepository) ClaimDue(ctx context.Context, batchSize int) ([]PushOutboxJob, error) {
	if batchSize <= 0 {
		return nil, nil
	}
	claimToken := uuid.NewString()
	result, err := r.DB.ExecContext(ctx, `
		UPDATE ALUMNI_PUSH_OUTBOX
		SET STATUS = 'PROCESSING',
		    CLAIM_TOKEN = ?,
		    UPDATED_AT = NOW()
		WHERE STATUS IN ('PENDING', 'FAILED')
		  AND NEXT_ATTEMPT_AT <= NOW()
		ORDER BY NEXT_ATTEMPT_AT ASC, PO_SEQ ASC
		LIMIT ?
	`, claimToken, batchSize)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, nil
	}

	var jobs []PushOutboxJob
	err = r.DB.SelectContext(ctx, &jobs, `
		SELECT PO_SEQ, EVENT_TYPE, EVENT_ID, USR_SEQ, MDT_SEQ, DEVICE_TOKEN, APNS_ENVIRONMENT,
		       BUNDLE_ID, TITLE, BODY, PAYLOAD_JSON, STATUS, ATTEMPT_COUNT, NEXT_ATTEMPT_AT,
		       LAST_ERROR_CODE, LAST_ERROR_MESSAGE, CREATED_AT, UPDATED_AT, SENT_AT
		FROM ALUMNI_PUSH_OUTBOX
		WHERE CLAIM_TOKEN = ?
		ORDER BY NEXT_ATTEMPT_AT ASC, PO_SEQ ASC
	`, claimToken)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *PushOutboxRepository) MarkSent(ctx context.Context, poSeq int) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE ALUMNI_PUSH_OUTBOX
		SET STATUS = 'SENT',
		    CLAIM_TOKEN = NULL,
		    LAST_ERROR_CODE = NULL,
		    LAST_ERROR_MESSAGE = NULL,
		    SENT_AT = NOW(),
		    UPDATED_AT = NOW()
		WHERE PO_SEQ = ?
	`, poSeq)
	return err
}

func (r *PushOutboxRepository) MarkRetryScheduled(ctx context.Context, poSeq int, nextAttemptAt time.Time, errorCode string, errorMessage string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE ALUMNI_PUSH_OUTBOX
		SET STATUS = 'FAILED',
		    ATTEMPT_COUNT = ATTEMPT_COUNT + 1,
		    NEXT_ATTEMPT_AT = ?,
		    LAST_ERROR_CODE = ?,
		    LAST_ERROR_MESSAGE = ?,
		    CLAIM_TOKEN = NULL,
		    UPDATED_AT = NOW()
		WHERE PO_SEQ = ?
	`, nextAttemptAt, nullableOutboxString(errorCode), nullableOutboxString(errorMessage), poSeq)
	return err
}

func (r *PushOutboxRepository) MarkDead(ctx context.Context, poSeq int, errorCode string, errorMessage string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE ALUMNI_PUSH_OUTBOX
		SET STATUS = 'DEAD',
		    ATTEMPT_COUNT = ATTEMPT_COUNT + 1,
		    LAST_ERROR_CODE = ?,
		    LAST_ERROR_MESSAGE = ?,
		    CLAIM_TOKEN = NULL,
		    UPDATED_AT = NOW()
		WHERE PO_SEQ = ?
	`, nullableOutboxString(errorCode), nullableOutboxString(errorMessage), poSeq)
	return err
}

func (r *PushOutboxRepository) ResetStuckProcessing(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		return 0, nil
	}
	result, err := r.DB.ExecContext(ctx, `
		UPDATE ALUMNI_PUSH_OUTBOX
		SET STATUS = 'FAILED',
		    NEXT_ATTEMPT_AT = NOW(),
		    LAST_ERROR_CODE = 'STUCK_PROCESSING',
		    LAST_ERROR_MESSAGE = 'processing job recovered after timeout',
		    CLAIM_TOKEN = NULL,
		    UPDATED_AT = NOW()
		WHERE STATUS = 'PROCESSING'
		  AND UPDATED_AT < DATE_SUB(NOW(), INTERVAL ? SECOND)
	`, int(olderThan.Seconds()))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *PushOutboxRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := r.DB.GetContext(ctx, &count, `SELECT COUNT(*) FROM ALUMNI_PUSH_OUTBOX WHERE STATUS = ?`, status)
	return count, err
}

func nullableOutboxString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
