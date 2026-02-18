package repository

import (
	"database/sql"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AuthRepository struct {
	DB *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{DB: db}
}

func (r *AuthRepository) LookupLegacySession(sessionID string) (*model.AuthUser, error) {
	var user model.AuthUser
	err := r.DB.Get(&user, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS
		FROM WEO_MEMBER_LOG l
		JOIN WEO_MEMBER m ON l.USR_SEQ = m.USR_SEQ
		WHERE l.SESSIONID = ?
		  AND l.LOG_DATE > DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY l.LOG_DATE DESC
		LIMIT 1
	`, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByKakaoID(kakaoID string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS, m.USR_PHONE, m.USR_FN, m.USR_EMAIL, m.USR_NICK, m.USR_PHOTO
		FROM WEO_MEMBER_SOCIAL s
		JOIN WEO_MEMBER m ON s.USR_SEQ = m.USR_SEQ
		WHERE s.NMS_GATE = 'KT' AND s.NMS_ID = ?
		LIMIT 1
	`, kakaoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByNamePhone(name string, phone string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_NAME = ? AND USR_PHONE = ? AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, name, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByFNName(fn string, name string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_FN = ? AND USR_NAME = ? AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, fn, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) InsertSocialLink(usrSeq int, gate string, socialID string, name string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER_SOCIAL (USR_SEQ, NMS_GATE, NMS_ID, NMS_NAME, REG_DATE)
		VALUES (?, ?, ?, ?, NOW())
	`, usrSeq, gate, socialID, name)
	return err
}

func (r *AuthRepository) InsertLoginLog(usrSeq int, sessionID string, ipAddr string, userAgent string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER_LOG
			(USR_SEQ, LOG_DATE, REG_DATE, REG_IPADDR, SESSIONID, REG_AGENT)
		VALUES (?, NOW(), NOW(), ?, ?, ?)
	`, usrSeq, ipAddr, sessionID, userAgent)
	return err
}

func (r *AuthRepository) UpdateLastLogin(usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET TOTAL_LOG_CNT = TOTAL_LOG_CNT + 1, LAST_LOG_DATE = NOW()
		WHERE USR_SEQ = ?
	`, usrSeq)
	return err
}

func (r *AuthRepository) FindMemberByLogin(usrID string, hashedPwd string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_ID = ? AND USR_PWD = ? AND USR_STATUS >= 'CCC'
		LIMIT 1
	`, usrID, hashedPwd)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByPhone(phone string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_PHONE = ? AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) InsertMember(usrID, name, phone, fn, email string) (int, error) {
	result, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT)
		VALUES (?, ?, ?, ?, ?, 'BBB', '', NOW(), 0)
	`, usrID, name, phone, fn, email)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *AuthRepository) GetMemberBySeq(usrSeq int) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		LIMIT 1
	`, usrSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
