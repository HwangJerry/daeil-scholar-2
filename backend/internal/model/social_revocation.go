// social_revocation.go — The durable outbox queue for asynchronous provider
// credential revocation (Kakao unlink, Apple revoke).
package model

import "time"

// SocialRevocationOutboxEntry mirrors a row of ALUMNI_SOCIAL_REVOCATION_OUTBOX,
// queued by ReserveSocialDisconnect (ACTION=DISCONNECT). The worker also
// understands legacy ACTION=ACCOUNT_DELETE rows created by the earlier account
// deletion flow. Both actions represent upstream revocation calls (Kakao
// unlink, Apple revoke) processed asynchronously by the worker.
type SocialRevocationOutboxEntry struct {
	OutboxID      int64     `db:"OUTBOX_ID"`
	USRSeq        int       `db:"USR_SEQ"`
	Provider      string    `db:"PROVIDER"`
	Action        string    `db:"ACTION"`
	Status        string    `db:"STATUS"`
	AttemptCount  int       `db:"ATTEMPT_COUNT"`
	NextAttemptAt time.Time `db:"NEXT_ATTEMPT_AT"`
	LastError     string    `db:"LAST_ERROR"`
	CreatedAt     time.Time `db:"CREATED_AT"`
	UpdatedAt     time.Time `db:"UPDATED_AT"`
}
