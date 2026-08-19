package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

const personalDonationWhere = `
	WHERE O_ACCOUNT_USR_SEQ = ?
	  AND O_TYPE = 'A'
	  AND O_LIFECYCLE_STATUS IN ('completed', 'partially_refunded')`

type PersonalDonationRepository struct {
	DB *sqlx.DB
}

func NewPersonalDonationRepository(db *sqlx.DB) *PersonalDonationRepository {
	return &PersonalDonationRepository{DB: db}
}

func (r *PersonalDonationRepository) GetTotals(usrSeq int) (int64, int, error) {
	var totals struct {
		TotalAmount int64 `db:"TOTAL_AMOUNT"`
		TotalCount  int   `db:"TOTAL_COUNT"`
	}
	err := r.DB.Get(&totals, `
		SELECT
			CAST(COALESCE(SUM(O_NET_RECEIVED_AMOUNT), 0) AS SIGNED) AS TOTAL_AMOUNT,
			CAST(COUNT(*) AS SIGNED) AS TOTAL_COUNT
		FROM WEO_ORDER`+personalDonationWhere, usrSeq)
	if err != nil {
		return 0, 0, err
	}
	return totals.TotalAmount, totals.TotalCount, nil
}

func (r *PersonalDonationRepository) List(usrSeq int, sort string, page, size int) ([]model.PersonalDonationItem, error) {
	orderBy := "ORDER BY O_DONATION_DATE DESC, O_SEQ DESC"
	if sort == "amount" {
		orderBy = "ORDER BY O_NET_RECEIVED_AMOUNT DESC, O_SEQ DESC"
	}

	items := make([]model.PersonalDonationItem, 0)
	err := r.DB.Select(&items, `
		SELECT
			O_SEQ AS ORDER_SEQ,
			COALESCE(DATE_FORMAT(O_DONATION_DATE, '%Y-%m-%d'), '') AS DONATION_DATE,
			COALESCE(O_GROSS_AMOUNT, 0) AS GROSS_AMOUNT,
			COALESCE(O_REFUNDED_AMOUNT, 0) AS REFUNDED_AMOUNT,
			COALESCE(O_NET_RECEIVED_AMOUNT, 0) AS NET_RECEIVED_AMOUNT,
			O_LIFECYCLE_STATUS AS LIFECYCLE_STATUS,
			COALESCE(O_PAYMENT_METHOD, 'other') AS PAYMENT_METHOD,
			COALESCE(O_SOURCE, 'other') AS SOURCE
		FROM WEO_ORDER`+personalDonationWhere+`
		`+orderBy+`
		LIMIT ? OFFSET ?
	`, usrSeq, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	return items, nil
}
