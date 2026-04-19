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

func (r *AuthRepository) FindMemberBySocialID(gate string, socialID string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS, m.USR_PHONE, m.USR_FN, m.USR_EMAIL, m.USR_NICK, m.USR_PHOTO
		FROM WEO_MEMBER_SOCIAL s
		JOIN WEO_MEMBER m ON s.USR_SEQ = m.USR_SEQ
		WHERE s.NMS_GATE = ? AND s.NMS_ID = ?
		LIMIT 1
	`, gate, socialID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByKakaoID(kakaoID string) (*model.User, error) {
	return r.FindMemberBySocialID("KT", kakaoID)
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

func (r *AuthRepository) InsertSocialLink(usrSeq int, gate string, socialID string, email string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER_SOCIAL (USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, REG_DATE)
		VALUES (?, ?, ?, ?, NOW())
	`, usrSeq, gate, socialID, email)
	return err
}

func (r *AuthRepository) DeleteLegacySessionsByUser(usrSeq int) error {
	_, err := r.DB.Exec(`DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?`, usrSeq)
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
		WHERE USR_ID = ? AND USR_PWD = ? AND USR_STATUS IN ('CCC', 'ZZZ')
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

func (r *AuthRepository) InsertMember(usrID, name, phone, fn, email, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic string) (int, error) {
	phonePublic := usrPhonePublic
	if phonePublic == "" {
		phonePublic = "Y"
	}
	emailPublic := usrEmailPublic
	if emailPublic == "" {
		emailPublic = "Y"
	}
	result, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', '', NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?)
	`, usrID, name, phone, fn, email, fmDept, jobCat, bizName, bizDesc, bizAddr, position, phonePublic, emailPublic)
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

func (r *AuthRepository) CheckIDExists(usrID string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_ID = ?`, usrID)
	return count > 0, err
}

func (r *AuthRepository) CheckPhoneExists(phone string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE = ?`, phone)
	return count > 0, err
}

func (r *AuthRepository) CheckEmailExists(email string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_EMAIL = ?`, email)
	return count > 0, err
}

func (r *AuthRepository) InsertMemberWithPwd(req model.RegisterRequest, hashedPwd string) (int, error) {
	phonePublic := req.USRPhonePublic
	if phonePublic == "" {
		phonePublic = "Y"
	}
	emailPublic := req.USREmailPublic
	if emailPublic == "" {
		emailPublic = "Y"
	}
	result, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_NICK, USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', ?, NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.UsrID, req.Name, req.Phone, req.FN, req.Email, hashedPwd,
		req.Nick, req.FmDept, req.JobCat, req.BizName, req.BizDesc, req.BizAddr,
		req.Position, phonePublic, emailPublic)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *AuthRepository) FindMemberByIDAndPwdAny(usrID, hashedPwd string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_ID = ? AND USR_PWD = ?
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
