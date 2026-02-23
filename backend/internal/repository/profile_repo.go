package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type ProfileRepository struct {
	DB *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) *ProfileRepository {
	return &ProfileRepository{DB: db}
}

func (r *ProfileRepository) GetProfile(usrSeq int) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.DB.QueryRow(`
		SELECT m.USR_SEQ, m.USR_NAME, IFNULL(m.USR_NICK, '') AS USR_NICK,
			IFNULL(m.USR_PHONE, '') AS USR_PHONE, IFNULL(m.USR_EMAIL, '') AS USR_EMAIL,
			IFNULL(m.USR_FN, '') AS USR_FN, IFNULL(m.USR_PHOTO, '') AS USR_PHOTO,
			IFNULL(m.USR_BIZ_NAME, '') AS USR_BIZ_NAME,
			IFNULL(m.USR_BIZ_DESC, '') AS USR_BIZ_DESC,
			IFNULL(m.USR_BIZ_ADDR, '') AS USR_BIZ_ADDR,
			IFNULL(m.USR_JOB_CAT, 0) AS USR_JOB_CAT,
			IFNULL(jc.AJC_NAME, '') AS AJC_NAME,
			IFNULL(jc.AJC_COLOR, '') AS AJC_COLOR,
			IFNULL(f.FM_DEPT, '') AS FM_DEPT,
			IFNULL(DATE_FORMAT(m.REG_DATE, '%Y. %m'), '') AS REG_DATE_FMT
		FROM WEO_MEMBER m
		LEFT JOIN FUNDAMENTAL_MEMBER f ON m.USR_SEQ = f.FM_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE m.USR_SEQ = ?
		LIMIT 1
	`, usrSeq).Scan(
		&profile.USRSeq, &profile.USRName, &profile.USRNick,
		&profile.USRPhone, &profile.USREmail, &profile.USRFN, &profile.USRPhoto,
		&profile.BizName, &profile.BizDesc, &profile.BizAddr,
		&profile.JobCat, &profile.JobCatName, &profile.JobCatColor,
		&profile.FmDept, &profile.RegDate,
	)
	if err != nil {
		return nil, err
	}

	// Load user tags
	tags, err := r.GetUserTags(usrSeq)
	if err != nil {
		return nil, err
	}
	profile.Tags = tags

	return &profile, nil
}

// GetUserTags returns the tags for a specific user.
func (r *ProfileRepository) GetUserTags(usrSeq int) ([]string, error) {
	var tags []string
	err := r.DB.Select(&tags, `
		SELECT AUT_TAG FROM ALUMNI_USER_TAG
		WHERE USR_SEQ = ?
		ORDER BY AUT_INDX ASC
	`, usrSeq)
	if err != nil {
		return []string{}, nil
	}
	if tags == nil {
		return []string{}, nil
	}
	return tags, nil
}

func (r *ProfileRepository) UpdateProfile(usrSeq int, req model.ProfileUpdateRequest) error {
	jobCat := 0
	if req.JobCat != nil {
		jobCat = *req.JobCat
	}
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET USR_NICK = ?, USR_PHONE = ?, USR_EMAIL = ?,
			USR_BIZ_NAME = ?, USR_BIZ_DESC = ?, USR_BIZ_ADDR = ?,
			USR_JOB_CAT = NULLIF(?, 0)
		WHERE USR_SEQ = ?
	`, req.USRNick, req.USRPhone, req.USREmail,
		req.BizName, req.BizDesc, req.BizAddr,
		jobCat, usrSeq)
	return err
}

// SaveUserTags replaces all tags for a user.
func (r *ProfileRepository) SaveUserTags(usrSeq int, tags []string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM ALUMNI_USER_TAG WHERE USR_SEQ = ?`, usrSeq); err != nil {
		return err
	}

	for i, tag := range tags {
		tag = trimTag(tag)
		if tag == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO ALUMNI_USER_TAG (USR_SEQ, AUT_TAG, AUT_INDX, REG_DATE)
			VALUES (?, ?, ?, NOW())
		`, usrSeq, tag, i); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CheckUserExists returns true if a user with the given USR_SEQ exists.
func (r *ProfileRepository) CheckUserExists(usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ = ?`, usrSeq)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func trimTag(s string) string {
	result := []byte{}
	for _, b := range []byte(s) {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			result = append(result, b)
		}
	}
	return string(result)
}
