package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type ProfileRepository struct {
	DB *sqlx.DB
}

type alumniVerificationAcademicRecord struct {
	Status                 model.VerificationStatus `db:"STATUS"`
	ApprovedGraduationYear sql.NullInt64            `db:"APPROVED_GRADUATION_YEAR"`
	ApprovedCohort         sql.NullString           `db:"APPROVED_COHORT"`
	ApprovedDepartment     sql.NullString           `db:"APPROVED_DEPARTMENT"`
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
			IFNULL(m.USR_POSITION, '') AS USR_POSITION,
			IFNULL(m.USR_JOB_CAT, 0) AS USR_JOB_CAT,
			IFNULL(jc.AJC_NAME, '') AS AJC_NAME,
			IFNULL(m.USR_DEPT, '') AS USR_DEPT,
			IFNULL(DATE_FORMAT(m.REG_DATE, '%Y. %m'), '') AS REG_DATE_FMT,
			IFNULL(m.USR_PHONE_PUBLIC, 'Y') AS USR_PHONE_PUBLIC,
			IFNULL(m.USR_EMAIL_PUBLIC, 'Y') AS USR_EMAIL_PUBLIC,
			IFNULL(m.USR_BIZ_CARD, '') AS USR_BIZ_CARD,
			CASE WHEN IFNULL(m.USR_PWD, '') != '' THEN 1 ELSE 0 END AS HAS_PASSWORD,
		CASE WHEN EXISTS (
			SELECT 1 FROM WEO_MEMBER_SOCIAL s
			WHERE s.USR_SEQ = m.USR_SEQ AND s.NMS_GATE = 'KT'
		) THEN 1 ELSE 0 END AS HAS_SOCIAL_LOGIN
		FROM WEO_MEMBER m
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE m.USR_SEQ = ?
		LIMIT 1
	`, usrSeq).Scan(
		&profile.USRSeq, &profile.USRName, &profile.USRNick,
		&profile.USRPhone, &profile.USREmail, &profile.USRFN, &profile.USRPhoto,
		&profile.BizName, &profile.BizDesc, &profile.BizAddr, &profile.Position,
		&profile.JobCat, &profile.JobCatName,
		&profile.FmDept, &profile.RegDate,
		&profile.USRPhonePublic, &profile.USREmailPublic, &profile.USRBizCard,
		&profile.HasPassword, &profile.HasSocialLogin,
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

func (r *ProfileRepository) SubmitAlumniVerification(usrSeq int, req model.AlumniVerificationSubmissionRequest) error {
	err := r.submitAlumniVerificationOnce(usrSeq, req)
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && (mysqlErr.Number == 1062 || mysqlErr.Number == 1213) {
		return r.submitAlumniVerificationOnce(usrSeq, req)
	}
	return err
}

func (r *ProfileRepository) submitAlumniVerificationOnce(usrSeq int, req model.AlumniVerificationSubmissionRequest) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current alumniVerificationAcademicRecord
	err = tx.Get(&current, `
		SELECT STATUS, APPROVED_GRADUATION_YEAR, APPROVED_COHORT, APPROVED_DEPARTMENT
		FROM ALUMNI_VERIFICATION
		WHERE USR_SEQ = ?
		FOR UPDATE
	`, usrSeq)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	isNewApplication := errors.Is(err, sql.ErrNoRows)
	nextStatus := model.VerificationPending
	academicChanged := true
	if !isNewApplication {
		academicChanged = !current.ApprovedGraduationYear.Valid ||
			int(current.ApprovedGraduationYear.Int64) != req.GraduationYear ||
			!current.ApprovedCohort.Valid || current.ApprovedCohort.String != req.Cohort ||
			!current.ApprovedDepartment.Valid || current.ApprovedDepartment.String != req.Department
		nextStatus = current.Status.AfterAcademicSubmission(academicChanged)
	}

	if _, err := tx.Exec(`
		UPDATE WEO_MEMBER
		SET USR_FN = ?, USR_DEPT = ?
		WHERE USR_SEQ = ?
	`, req.Cohort, req.Department, usrSeq); err != nil {
		return err
	}
	if isNewApplication {
		if _, err := tx.Exec(`
			INSERT INTO ALUMNI_VERIFICATION (
				USR_SEQ, STATUS, GRADUATION_YEAR, COHORT, DEPARTMENT,
				REJECTION_REASON, SUBMITTED_AT, REVIEWED_AT, REVIEWED_BY,
				APPROVED_GRADUATION_YEAR, APPROVED_COHORT, APPROVED_DEPARTMENT,
				CREATED_AT, UPDATED_AT
			) VALUES (?, ?, ?, ?, ?, NULL, NOW(), NULL, NULL, NULL, NULL, NULL, NOW(), NOW())
		`, usrSeq, model.VerificationPending, req.GraduationYear, req.Cohort, req.Department); err != nil {
			return err
		}
	} else if current.Status == model.VerificationApproved && !academicChanged {
		if _, err := tx.Exec(`
			UPDATE ALUMNI_VERIFICATION
			SET GRADUATION_YEAR = ?, COHORT = ?, DEPARTMENT = ?, UPDATED_AT = NOW()
			WHERE USR_SEQ = ?
		`, req.GraduationYear, req.Cohort, req.Department, usrSeq); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(`
			UPDATE ALUMNI_VERIFICATION
			SET STATUS = ?, GRADUATION_YEAR = ?, COHORT = ?, DEPARTMENT = ?,
				REJECTION_REASON = NULL, SUBMITTED_AT = NOW(), REVIEWED_AT = NULL,
				REVIEWED_BY = NULL, UPDATED_AT = NOW()
			WHERE USR_SEQ = ?
		`, nextStatus, req.GraduationYear, req.Cohort, req.Department, usrSeq); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ProfileRepository) GetAlumniVerification(usrSeq int) (*model.AlumniVerification, error) {
	var verification model.AlumniVerification
	var graduationYear sql.NullInt64
	var cohort, department, rejectionReason sql.NullString
	var submittedAt, reviewedAt sql.NullTime
	err := r.DB.QueryRow(`
		SELECT STATUS, GRADUATION_YEAR, COHORT, DEPARTMENT, REJECTION_REASON,
			SUBMITTED_AT, REVIEWED_AT
		FROM ALUMNI_VERIFICATION
		WHERE USR_SEQ = ?
		LIMIT 1
	`, usrSeq).Scan(
		&verification.Status, &graduationYear, &cohort, &department, &rejectionReason,
		&submittedAt, &reviewedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &model.AlumniVerification{Status: model.VerificationUnsubmitted}, nil
	}
	if err != nil {
		return nil, err
	}
	if graduationYear.Valid {
		value := int(graduationYear.Int64)
		verification.GraduationYear = &value
	}
	if cohort.Valid {
		verification.Cohort = &cohort.String
	}
	if department.Valid {
		verification.Department = &department.String
	}
	if rejectionReason.Valid {
		verification.RejectionReason = &rejectionReason.String
	}
	if submittedAt.Valid {
		verification.SubmittedAt = &submittedAt.Time
	}
	if reviewedAt.Valid {
		verification.ReviewedAt = &reviewedAt.Time
	}
	return &verification, nil
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
	phonePublic := req.USRPhonePublic
	if phonePublic == "" {
		phonePublic = "Y"
	}
	emailPublic := req.USREmailPublic
	if emailPublic == "" {
		emailPublic = "Y"
	}
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET USR_NAME = ?, USR_PHONE = ?, USR_EMAIL = ?,
			USR_BIZ_NAME = ?, USR_BIZ_DESC = ?, USR_BIZ_ADDR = ?,
			USR_POSITION = NULLIF(?, ''),
			USR_JOB_CAT = NULLIF(?, 0),
			USR_PHONE_PUBLIC = ?, USR_EMAIL_PUBLIC = ?
		WHERE USR_SEQ = ?
	`, req.USRName, req.USRPhone, req.USREmail,
		req.BizName, req.BizDesc, req.BizAddr,
		req.Position, jobCat, phonePublic, emailPublic, usrSeq)
	return err
}

// UpdateProfilePhoto updates only the USR_PHOTO column for a user.
func (r *ProfileRepository) UpdateProfilePhoto(usrSeq int, url string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_PHOTO = ? WHERE USR_SEQ = ?`, url, usrSeq)
	return err
}

// UpdateBizCard updates only the USR_BIZ_CARD column for a user.
func (r *ProfileRepository) UpdateBizCard(usrSeq int, url string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_BIZ_CARD = ? WHERE USR_SEQ = ?`, url, usrSeq)
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

// GetPasswordHash returns the stored USR_PWD hash for a user.
func (r *ProfileRepository) GetPasswordHash(usrSeq int) (string, error) {
	var pwd string
	err := r.DB.QueryRow(`SELECT IFNULL(USR_PWD, '') FROM WEO_MEMBER WHERE USR_SEQ = ? LIMIT 1`, usrSeq).Scan(&pwd)
	return pwd, err
}

// UpdatePassword sets a new USR_PWD hash for a user.
func (r *ProfileRepository) UpdatePassword(usrSeq int, hashedPwd string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_PWD = ? WHERE USR_SEQ = ?`, hashedPwd, usrSeq)
	return err
}

// CheckUserExists returns true if a user with the given USR_SEQ exists.
func (r *ProfileRepository) CheckUserExists(usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_SEQ = ? AND USR_STATUS != 'AAA'`, usrSeq)
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
