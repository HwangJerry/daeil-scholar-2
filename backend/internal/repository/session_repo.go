package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type SessionRepository struct {
	DB *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{DB: db}
}

func (r *SessionRepository) CreateSession(session model.UserSession) error {
	_, err := r.DB.Exec(`
		INSERT INTO USER_SESSION (SESSION_ID, USR_SEQ, PROVIDER, EXPIRES_AT, CREATED_AT)
		VALUES (?, ?, ?, ?, ?)
	`, session.SessionID, session.USRSeq, session.Provider, session.ExpiresAt, session.CreatedAt)
	return err
}

func (r *SessionRepository) DeleteExpiredSessions() (int64, error) {
	result, err := r.DB.Exec(`
		DELETE FROM USER_SESSION WHERE EXPIRES_AT < NOW()
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
