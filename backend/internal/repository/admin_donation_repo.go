// Admin donation repository — config update and snapshot history queries
package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminDonationRepository struct {
	DB *sqlx.DB
}

func NewAdminDonationRepository(db *sqlx.DB) *AdminDonationRepository {
	return &AdminDonationRepository{DB: db}
}

func (r *AdminDonationRepository) UpdateConfig(goal int64, manualAdj int64, note string, operSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE DONATION_CONFIG
		SET DC_GOAL = ?, DC_MANUAL_ADJ = ?, DC_NOTE = ?, REG_OPER = ?, REG_DATE = NOW()
		WHERE IS_ACTIVE = 'Y'
	`, goal, manualAdj, note, operSeq)
	return err
}

func (r *AdminDonationRepository) GetSnapshotHistory(days int) ([]model.DonationSnapshot, error) {
	var snapshots []model.DonationSnapshot
	err := r.DB.Select(&snapshots, `
		SELECT DS_SEQ, DS_DATE, DS_TOTAL, DS_MANUAL_ADJ, DS_DONOR_CNT, DS_GOAL,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE
		FROM DONATION_SNAPSHOT
		ORDER BY DS_DATE DESC
		LIMIT ?
	`, days)
	return snapshots, err
}

func (r *AdminDonationRepository) GetDonationOrders(page, size int, search, status, payType string) ([]model.AdminDonationOrderRow, int, error) {
	args := []interface{}{}
	conditions := []string{"o.O_TYPE = 'A'"}
	if search != "" {
		conditions = append(conditions, "m.USR_NAME LIKE ?")
		args = append(args, search+"%")
	}
	if status != "" {
		conditions = append(conditions, "o.O_PAYMENT = ?")
		args = append(args, status)
	}
	if payType != "" {
		conditions = append(conditions, "o.O_PAY_TYPE = ?")
		args = append(args, payType)
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.DB.Get(&total, "SELECT COUNT(*) FROM WEO_ORDER o JOIN WEO_MEMBER m ON o.USR_SEQ = m.USR_SEQ "+where, countArgs...); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	query := `SELECT o.O_SEQ, o.USR_SEQ, m.USR_NAME, o.O_PRICE, o.O_PAY_TYPE, o.O_PAYMENT,
		       IFNULL(DATE_FORMAT(o.O_PAYDATE,'%Y-%m-%d %H:%i:%s'),'') AS PAY_DATE,
		       IFNULL(o.O_GATE,'') AS O_GATE,
		       IFNULL(DATE_FORMAT(o.O_REGDATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE
		FROM WEO_ORDER o JOIN WEO_MEMBER m ON o.USR_SEQ = m.USR_SEQ ` + where + ` ORDER BY o.O_SEQ DESC LIMIT ? OFFSET ?`
	args = append(args, size, offset)

	var rows []model.AdminDonationOrderRow
	if err := r.DB.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AdminDonationRepository) UpdateDonationOrder(seq int, payment string, amount int) error {
	_, err := r.DB.Exec(`UPDATE WEO_ORDER SET O_PAYMENT = ?, O_PAY = ?, EDT_DATE = NOW() WHERE O_SEQ = ? AND O_TYPE = 'A'`,
		payment, amount, seq)
	return err
}
