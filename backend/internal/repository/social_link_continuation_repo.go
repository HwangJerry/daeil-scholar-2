package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

var (
	ErrSocialLinkTokenInvalid  = errors.New("social link token is invalid or expired")
	ErrSocialLinkTokenConsumed = errors.New("social link token already consumed")
	ErrSocialLinkReauth        = errors.New("social link account reauthentication failed")
	ErrSocialLinkReauthLocked  = errors.New("social link account reauthentication locked")
	ErrSocialIdentityOwner     = errors.New("social identity belongs to another member")
)

const socialLinkMaxReauthFailures = 5

func (r *AuthRepository) InsertSocialLinkContinuation(
	tokenHash string,
	provider string,
	subject string,
	email string,
	expiresAt time.Time,
) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD
			(SLR_PROVIDER, SLR_SUBJECT, SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT, SLR_EXPIRES_AT, SLR_UPDATED_AT)
		VALUES (?, ?, 0, NULL, ?, NOW())
		ON DUPLICATE KEY UPDATE
			SLR_FAILED_ATTEMPTS = IF(SLR_EXPIRES_AT <= NOW(), 0, SLR_FAILED_ATTEMPTS),
			SLR_LOCKED_AT = IF(SLR_EXPIRES_AT <= NOW(), NULL, SLR_LOCKED_AT),
			SLR_EXPIRES_AT = GREATEST(SLR_EXPIRES_AT, VALUES(SLR_EXPIRES_AT)),
			SLR_UPDATED_AT = NOW()
	`, provider, subject, expiresAt); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_SOCIAL_LINK_CONTINUATION
			(SLC_TOKEN_HASH, SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT, SLC_CREATED_AT)
		VALUES (?, ?, ?, NULLIF(?, ''), 'READY', ?, NOW())
	`, tokenHash, provider, subject, email, expiresAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AuthRepository) CompleteSocialLinkContinuation(
	tokenHash string,
	email string,
	hashedPassword string,
) (*model.User, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var continuation struct {
		Provider  string         `db:"SLC_PROVIDER"`
		Subject   string         `db:"SLC_SUBJECT"`
		Email     sql.NullString `db:"SLC_EMAIL"`
		Status    string         `db:"SLC_STATUS"`
		ExpiresAt time.Time      `db:"SLC_EXPIRES_AT"`
	}
	err = tx.Get(&continuation, `
		SELECT SLC_PROVIDER, SLC_SUBJECT, SLC_EMAIL, SLC_STATUS, SLC_EXPIRES_AT
		FROM ALUMNI_SOCIAL_LINK_CONTINUATION
		WHERE SLC_TOKEN_HASH = ?
		FOR UPDATE
	`, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSocialLinkTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	if continuation.Status == "CONSUMED" {
		return nil, ErrSocialLinkTokenConsumed
	}
	if continuation.Status != "READY" || !continuation.ExpiresAt.After(time.Now()) {
		return nil, ErrSocialLinkTokenInvalid
	}
	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_SOCIAL_LINK_REAUTH_GUARD
			(SLR_PROVIDER, SLR_SUBJECT, SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT, SLR_EXPIRES_AT, SLR_UPDATED_AT)
		VALUES (?, ?, 0, NULL, ?, NOW())
		ON DUPLICATE KEY UPDATE
			SLR_FAILED_ATTEMPTS = IF(SLR_EXPIRES_AT <= NOW(), 0, SLR_FAILED_ATTEMPTS),
			SLR_LOCKED_AT = IF(SLR_EXPIRES_AT <= NOW(), NULL, SLR_LOCKED_AT),
			SLR_EXPIRES_AT = GREATEST(SLR_EXPIRES_AT, VALUES(SLR_EXPIRES_AT)),
			SLR_UPDATED_AT = NOW()
	`, continuation.Provider, continuation.Subject, continuation.ExpiresAt); err != nil {
		return nil, err
	}
	var guard struct {
		FailedAttempts int          `db:"SLR_FAILED_ATTEMPTS"`
		LockedAt       sql.NullTime `db:"SLR_LOCKED_AT"`
	}
	if err := tx.Get(&guard, `
		SELECT SLR_FAILED_ATTEMPTS, SLR_LOCKED_AT
		FROM ALUMNI_SOCIAL_LINK_REAUTH_GUARD
		WHERE SLR_PROVIDER = ? AND SLR_SUBJECT = ?
		FOR UPDATE
	`, continuation.Provider, continuation.Subject); err != nil {
		return nil, err
	}
	if guard.LockedAt.Valid || guard.FailedAttempts >= socialLinkMaxReauthFailures {
		return nil, ErrSocialLinkReauthLocked
	}

	var users []model.User
	if err := tx.Select(&users, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE LOWER(TRIM(USR_EMAIL)) = LOWER(TRIM(?))
		  AND USR_PWD = ?
		  AND USR_STATUS IN ('BAA','BBB','CCC','ZZZ')
		LIMIT 2
	`, email, hashedPassword); err != nil {
		return nil, err
	}
	if len(users) != 1 {
		result, err := tx.Exec(`
			UPDATE ALUMNI_SOCIAL_LINK_REAUTH_GUARD
			SET SLR_FAILED_ATTEMPTS = SLR_FAILED_ATTEMPTS + 1,
			    SLR_LOCKED_AT = CASE
			        WHEN SLR_FAILED_ATTEMPTS >= ? THEN COALESCE(SLR_LOCKED_AT, NOW())
			        ELSE SLR_LOCKED_AT
			    END
			WHERE SLR_PROVIDER = ?
			  AND SLR_SUBJECT = ?
			  AND SLR_LOCKED_AT IS NULL
		`, socialLinkMaxReauthFailures, continuation.Provider, continuation.Subject)
		if err != nil {
			return nil, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected < 1 {
			return nil, ErrSocialLinkReauthLocked
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		if guard.FailedAttempts+1 >= socialLinkMaxReauthFailures {
			return nil, ErrSocialLinkReauthLocked
		}
		return nil, ErrSocialLinkReauth
	}
	user := users[0]

	var owner int
	err = tx.Get(&owner, `
		SELECT USR_SEQ
		FROM WEO_MEMBER_SOCIAL
		WHERE NMS_GATE = ? AND NMS_ID = ?
		FOR UPDATE
	`, continuation.Provider, continuation.Subject)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		providerEmail := ""
		if continuation.Email.Valid {
			providerEmail = continuation.Email.String
		}
		if _, err := tx.Exec(`
			INSERT INTO WEO_MEMBER_SOCIAL
				(USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, NMS_STATUS, NMS_EMAIL_ENABLED, REG_DATE)
			VALUES (?, ?, ?, NULLIF(?, ''), 'ACTIVE', 'Y', NOW())
		`, user.USRSeq, continuation.Provider, continuation.Subject, providerEmail); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case owner != user.USRSeq:
		return nil, ErrSocialIdentityOwner
	}

	result, err := tx.Exec(`
		UPDATE ALUMNI_SOCIAL_LINK_CONTINUATION
		SET SLC_STATUS = 'CONSUMED', SLC_CONSUMED_AT = NOW()
		WHERE SLC_TOKEN_HASH = ? AND SLC_STATUS = 'READY'
	`, tokenHash)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, ErrSocialLinkTokenConsumed
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &user, nil
}
