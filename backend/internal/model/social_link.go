// social_link.go — Data held server-side during the social account linking flow
package model

import "time"

// SocialLinkData holds social provider user info stored server-side during the account linking flow.
type SocialLinkData struct {
	Provider        string // "KT" (Kakao), "NV" (Naver), "FB" (Facebook)
	SocialID        string
	Email           string
	Nickname        string
	ProfileImageURL string // empty when the provider did not supply an image (e.g. consent declined)
	AccessToken     string
	RevocationToken string
}

// KakaoLinkData is kept as an alias for backward compatibility with cached data.
type KakaoLinkData = SocialLinkData

// SocialRevocationOutboxEntry mirrors a row of ALUMNI_SOCIAL_REVOCATION_OUTBOX,
// queued whenever a provider disconnect/account deletion needs an upstream
// revocation call (Kakao unlink, Apple revoke) processed asynchronously.
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
