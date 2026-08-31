// Admin member repository — member list, detail, and status update queries
package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrVerificationStale         = errors.New("verification stale")
	ErrVerificationStateConflict = errors.New("verification state conflict")
)

type AdminMemberRepository struct {
	DB *sqlx.DB
}

func NewAdminMemberRepository(db *sqlx.DB) *AdminMemberRepository {
	return &AdminMemberRepository{DB: db}
}

func (r *AdminMemberRepository) GetMembers(page, size int, q, fn, status string) ([]model.AdminMemberRow, int, error) {
	args := []interface{}{}
	conditions := []string{}
	if q != "" {
		conditions = append(conditions, "(USR_NAME LIKE ? OR USR_PHONE LIKE ?)")
		args = append(args, q+"%", q+"%")
	}
	if fn != "" {
		conditions = append(conditions, "USR_FN = ?")
		args = append(args, fn)
	}
	if status != "" {
		conditions = append(conditions, "USR_STATUS = ?")
		args = append(args, status)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.DB.Get(&total, "SELECT COUNT(*) FROM WEO_MEMBER "+where, countArgs...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	query := `SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS,
		       IFNULL(USR_FN,'') AS USR_FN, IFNULL(USR_PHONE,'') AS USR_PHONE,
		       IFNULL(USR_EMAIL,'') AS USR_EMAIL,
		       IFNULL(USR_DEPT,'') AS USR_DEPT,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE,
		       IFNULL(DATE_FORMAT(LAST_LOG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS VISIT_DATE
		FROM WEO_MEMBER ` + where + ` ORDER BY USR_SEQ DESC LIMIT ? OFFSET ?`
	args = append(args, size, offset)

	var rows []model.AdminMemberRow
	if err := r.DB.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AdminMemberRepository) GetMemberDetail(seq int) (*model.AdminMemberDetail, error) {
	var m model.AdminMemberDetail
	err := r.DB.Get(&m, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS,
		       IFNULL(USR_FN,'') AS USR_FN, IFNULL(USR_PHONE,'') AS USR_PHONE,
		       IFNULL(USR_EMAIL,'') AS USR_EMAIL, IFNULL(USR_NICK,'') AS USR_NICK,
		       IFNULL(USR_PHOTO,'') AS USR_PHOTO,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE,
		       TOTAL_LOG_CNT AS VISIT_CNT,
		       IFNULL(DATE_FORMAT(LAST_LOG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS VISIT_DATE
		FROM WEO_MEMBER WHERE USR_SEQ = ? LIMIT 1
	`, seq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *AdminMemberRepository) ListAlumniVerifications(status string) ([]model.AdminAlumniVerification, error) {
	rows := []model.AdminAlumniVerification{}
	err := r.DB.Select(&rows, `
		SELECT v.USR_SEQ, m.USR_NAME, v.STATUS, v.GRADUATION_YEAR, v.COHORT, v.DEPARTMENT,
			v.REJECTION_REASON, v.SUBMITTED_AT, v.REVIEWED_AT, v.UPDATED_AT
		FROM ALUMNI_VERIFICATION v
		JOIN WEO_MEMBER m ON m.USR_SEQ = v.USR_SEQ
		WHERE (? = '' OR v.STATUS = ?)
		ORDER BY v.SUBMITTED_AT ASC, v.USR_SEQ ASC
	`, status, status)
	return rows, err
}

func (r *AdminMemberRepository) GetAlumniVerificationDetail(usrSeq int) (*model.AdminAlumniVerification, error) {
	var detail model.AdminAlumniVerification
	err := r.DB.Get(&detail, `
		SELECT v.USR_SEQ, m.USR_NAME, v.STATUS, v.GRADUATION_YEAR, v.COHORT, v.DEPARTMENT,
			v.REJECTION_REASON, v.SUBMITTED_AT, v.REVIEWED_AT, v.UPDATED_AT
		FROM ALUMNI_VERIFICATION v
		JOIN WEO_MEMBER m ON m.USR_SEQ = v.USR_SEQ
		WHERE v.USR_SEQ = ?
		LIMIT 1
	`, usrSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (r *AdminMemberRepository) UpdateMemberStatus(seq int, status string) error {
	_, err := updateMemberStatusAndAdminRole(r.DB, seq, status)
	return err
}

// UpsertRootAdminMember creates or updates the legacy seed administrator and
// keeps its canonical root role in the same transaction.
func (r *AdminMemberRepository) UpsertRootAdminMember(usrID, passwordHash, name string) (int, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var existing struct {
		Seq    int            `db:"USR_SEQ"`
		Status sql.NullString `db:"USR_STATUS"`
		FN     string         `db:"USR_FN"`
		Dept   string         `db:"USR_DEPT"`
	}
	err = tx.Get(&existing, `
		SELECT USR_SEQ, USR_STATUS, IFNULL(USR_FN, '') AS USR_FN, IFNULL(USR_DEPT, '') AS USR_DEPT
		FROM WEO_MEMBER
		WHERE USR_ID = ?
		FOR UPDATE
	`, usrID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	usrSeq := existing.Seq
	if errors.Is(err, sql.ErrNoRows) {
		result, insertErr := tx.Exec(`
			INSERT INTO WEO_MEMBER (USR_ID, USR_PWD, USR_NAME, USR_STATUS, REG_DATE)
			VALUES (?, ?, ?, 'ZZZ', NOW())
		`, usrID, passwordHash, name)
		if insertErr != nil {
			return 0, insertErr
		}
		insertedID, insertErr := result.LastInsertId()
		if insertErr != nil {
			return 0, insertErr
		}
		usrSeq = int(insertedID)
		if err := syncAdminRoleForMemberInsertTx(tx, usrSeq, rootAdminMemberStatus); err != nil {
			return 0, err
		}
		if err := syncVerificationForMemberInsertTx(tx, usrSeq, rootAdminMemberStatus, "", ""); err != nil {
			return 0, err
		}
	} else {
		if _, err := tx.Exec(`
			UPDATE WEO_MEMBER
			SET USR_PWD = ?, USR_NAME = ?, USR_STATUS = 'ZZZ'
			WHERE USR_SEQ = ?
		`, passwordHash, name, usrSeq); err != nil {
			return 0, err
		}
		if err := syncAdminRoleForStatusChangeTx(tx, usrSeq, existing.Status, rootAdminMemberStatus); err != nil {
			return 0, err
		}
		if err := syncVerificationForStatusChangeTx(tx, usrSeq, existing.Status, rootAdminMemberStatus, existing.FN, existing.Dept); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return usrSeq, nil
}

func (r *AdminMemberRepository) RejectAlumniVerification(
	usrSeq int,
	reviewerSeq int,
	reason string,
	expectedUpdatedAt time.Time,
) error {
	result, err := r.DB.Exec(`
		UPDATE ALUMNI_VERIFICATION
		SET STATUS = ?, REJECTION_REASON = ?, REVIEWED_AT = NOW(), REVIEWED_BY = ?, UPDATED_AT = NOW()
		WHERE USR_SEQ = ?
			AND STATUS IN ('pending', 'reapproval_pending')
			AND UPDATED_AT = ?
	`, model.VerificationRejected, reason, reviewerSeq, usrSeq, expectedUpdatedAt)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		var currentStatus model.VerificationStatus
		var currentUpdatedAt time.Time
		if err := r.DB.QueryRow(`
			SELECT STATUS, UPDATED_AT
			FROM ALUMNI_VERIFICATION
			WHERE USR_SEQ = ?
		`, usrSeq).Scan(&currentStatus, &currentUpdatedAt); err != nil {
			return ErrVerificationStateConflict
		}
		if !currentUpdatedAt.Equal(expectedUpdatedAt) {
			return ErrVerificationStale
		}
		return ErrVerificationStateConflict
	}
	return nil
}

func (r *AdminMemberRepository) ApproveAlumniVerification(
	usrSeq int,
	reviewerSeq int,
	expectedUpdatedAt time.Time,
) error {
	result, err := r.DB.Exec(`
		UPDATE ALUMNI_VERIFICATION
		SET STATUS = ?, REJECTION_REASON = NULL,
			REVIEWED_AT = NOW(), REVIEWED_BY = ?,
			APPROVED_GRADUATION_YEAR = GRADUATION_YEAR,
			APPROVED_COHORT = COHORT,
			APPROVED_DEPARTMENT = DEPARTMENT,
			UPDATED_AT = NOW()
		WHERE USR_SEQ = ?
			AND STATUS IN ('pending', 'reapproval_pending')
			AND UPDATED_AT = ?
	`, model.VerificationApproved, reviewerSeq, usrSeq, expectedUpdatedAt)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		var currentStatus model.VerificationStatus
		var currentUpdatedAt time.Time
		if err := r.DB.QueryRow(`
			SELECT STATUS, UPDATED_AT
			FROM ALUMNI_VERIFICATION
			WHERE USR_SEQ = ?
		`, usrSeq).Scan(&currentStatus, &currentUpdatedAt); err != nil {
			return ErrVerificationStateConflict
		}
		if !currentUpdatedAt.Equal(expectedUpdatedAt) {
			return ErrVerificationStale
		}
		return ErrVerificationStateConflict
	}
	return nil
}

func (r *AdminMemberRepository) HasKakaoLink(usrSeq int) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = ? AND NMS_GATE = 'KT'`, usrSeq)
	return count > 0, err
}

func (r *AdminMemberRepository) CountTotalMembers() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_MEMBER`)
	return c, err
}

func (r *AdminMemberRepository) CountKakaoLinked() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(DISTINCT USR_SEQ) FROM WEO_MEMBER_SOCIAL WHERE NMS_GATE = 'KT'`)
	return c, err
}

func (r *AdminMemberRepository) CountRecentLogins(days int) (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(DISTINCT USR_SEQ) FROM WEO_MEMBER_LOG WHERE LOG_DATE > DATE_SUB(NOW(), INTERVAL ? DAY)`, days)
	return c, err
}

func (r *AdminMemberRepository) CountPendingMembers() (int, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_STATUS = 'BBB'`)
	return c, err
}
