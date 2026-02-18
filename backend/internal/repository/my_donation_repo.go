// my_donation_repo.go — queries for user donation history
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type MyDonationRepository struct {
	DB *sqlx.DB
}

func NewMyDonationRepository(db *sqlx.DB) *MyDonationRepository {
	return &MyDonationRepository{DB: db}
}

func (r *MyDonationRepository) GetMyDonations(usrSeq int, sort string, page int, size int) ([]model.MyDonationItem, int, error) {
	baseWhere := `WHERE USR_SEQ = ? AND O_TYPE = 'A' AND O_PAYMENT = 'Y'`

	var total int
	if err := r.DB.Get(&total, "SELECT COUNT(*) FROM WEO_ORDER "+baseWhere, usrSeq); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY O_PAYDATE DESC, O_SEQ DESC"
	if sort == "amount" {
		orderBy = "ORDER BY O_PRICE DESC, O_SEQ DESC"
	}

	offset := (page - 1) * size
	query := `SELECT O_SEQ AS OrderSeq, O_PRICE AS Amount, O_PAY_TYPE AS PayType,
		       IFNULL(DATE_FORMAT(O_PAYDATE,'%Y-%m-%d %H:%i:%s'),'') AS PaidAt
		FROM WEO_ORDER ` + baseWhere + ` ` + orderBy + ` LIMIT ? OFFSET ?`

	var items []model.MyDonationItem
	if err := r.DB.Select(&items, query, usrSeq, size, offset); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *MyDonationRepository) GetMyTotalDonation(usrSeq int) (int64, error) {
	var total int64
	err := r.DB.Get(&total, `SELECT IFNULL(SUM(O_PRICE), 0) FROM WEO_ORDER WHERE USR_SEQ = ? AND O_TYPE = 'A' AND O_PAYMENT = 'Y'`, usrSeq)
	return total, err
}
