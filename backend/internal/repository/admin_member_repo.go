// Admin member repository — member list, detail, and status update queries
package repository

import (
	"database/sql"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminMemberRepository struct {
	DB *sqlx.DB
}

func NewAdminMemberRepository(db *sqlx.DB) *AdminMemberRepository {
	return &AdminMemberRepository{DB: db}
}

func (r *AdminMemberRepository) GetMembers(page, size int, name, fn, status string) ([]model.AdminMemberRow, int, error) {
	args := []interface{}{}
	conditions := []string{}
	if name != "" {
		conditions = append(conditions, "USR_NAME LIKE ?")
		args = append(args, name+"%")
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

func (r *AdminMemberRepository) UpdateMemberStatus(seq int, status string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_STATUS = ? WHERE USR_SEQ = ?`, status, seq)
	return err
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
