package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type DonateRepository struct {
	DB *sqlx.DB
}

func NewDonateRepository(db *sqlx.DB) *DonateRepository {
	return &DonateRepository{DB: db}
}

// InsertOrder is a legacy-field-only EasyPay/in-app writer. Its entry routes and
// billing triggers are disabled (backend/cmd/server/routes.go:79, :158, :262 and
// backend/cmd/server/main.go:94). Before re-enabling them, make this writer populate
// the canonical donation fields used by personal_donation_repo.go, including
// O_ACCOUNT_USR_SEQ, O_LIFECYCLE_STATUS, and O_NET_RECEIVED_AMOUNT.
func (r *DonateRepository) InsertOrder(usrSeq int, gate string, payType string, price int, ip string) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO WEO_ORDER (USR_SEQ, O_GATE, O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR)
		VALUES (?, ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
	`, usrSeq, gate, payType, price, ip)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil || id == 0 {
		var lastID int64
		if qErr := r.DB.Get(&lastID, "SELECT LAST_INSERT_ID()"); qErr != nil {
			if err != nil {
				return 0, err
			}
			return 0, qErr
		}
		return lastID, nil
	}
	return id, nil
}

// InsertOrderTx creates a new WEO_ORDER row within an existing tx. It is a
// legacy-field-only writer for the currently disabled subscription billing flow
// (backend/cmd/server/main.go:94 and backend/cmd/server/routes.go:262). Before
// re-enabling that flow, make this writer populate the canonical donation fields used
// by personal_donation_repo.go, including O_ACCOUNT_USR_SEQ, O_LIFECYCLE_STATUS, and
// O_NET_RECEIVED_AMOUNT. Order creation, PG approval, and status update then remain in
// a single transaction per subscription charge.
func (r *DonateRepository) InsertOrderTx(tx *sqlx.Tx, usrSeq int, gate, payType string, price int, ip string) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO WEO_ORDER (USR_SEQ, O_GATE, O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR)
		VALUES (?, ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
	`, usrSeq, gate, payType, price, ip)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *DonateRepository) InsertPGData(data *model.PGData) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO WEO_PG_DATA (CNO, RES_CD, RES_MSG, AMOUNT, NUM_CARD, TRAN_DATE, AUTH_NO, PAY_TYPE, O_SEQ)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, data.CNO, data.ResCD, data.ResMsg, data.Amount, data.NumCard, data.TranDate, data.AuthNo, data.PayType, data.OSeq)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertPGDataTx inserts a PG approval record within an existing transaction.
func (r *DonateRepository) InsertPGDataTx(tx *sqlx.Tx, data *model.PGData) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO WEO_PG_DATA (CNO, RES_CD, RES_MSG, AMOUNT, NUM_CARD, TRAN_DATE, AUTH_NO, PAY_TYPE, O_SEQ)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, data.CNO, data.ResCD, data.ResMsg, data.Amount, data.NumCard, data.TranDate, data.AuthNo, data.PayType, data.OSeq)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateOrderPayment is a legacy-field-only EasyPay/in-app writer. Its entry routes
// and billing triggers are disabled (backend/cmd/server/routes.go:79, :158, :262 and
// backend/cmd/server/main.go:94). Before re-enabling them, make this writer populate
// the canonical donation fields used by personal_donation_repo.go, including
// O_ACCOUNT_USR_SEQ, O_LIFECYCLE_STATUS, and O_NET_RECEIVED_AMOUNT.
func (r *DonateRepository) UpdateOrderPayment(orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
	result, err := r.DB.Exec(`
		UPDATE WEO_ORDER
		SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
		    EDT_DATE = NOW(), EDT_IPADDR = ?
		WHERE O_SEQ = ? AND O_PAYMENT = 'N'
	`, amount, pgSeq, ip, orderSeq)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// UpdateOrderPaymentTx marks an order as paid within an existing transaction. It is a
// legacy-field-only writer used by the currently disabled EasyPay/in-app and
// subscription flows (backend/cmd/server/routes.go:79, :158, :262 and
// backend/cmd/server/main.go:94). Before re-enabling those flows, make this writer and
// subscription_billing.go populate the canonical donation fields used by
// personal_donation_repo.go, including O_ACCOUNT_USR_SEQ, O_LIFECYCLE_STATUS, and
// O_NET_RECEIVED_AMOUNT.
func (r *DonateRepository) UpdateOrderPaymentTx(tx *sqlx.Tx, orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
	result, err := tx.Exec(`
		UPDATE WEO_ORDER
		SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
		    EDT_DATE = NOW(), EDT_IPADDR = ?
		WHERE O_SEQ = ? AND O_PAYMENT = 'N'
	`, amount, pgSeq, ip, orderSeq)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *DonateRepository) GetOrderGate(orderSeq int) (string, error) {
	var gate string
	err := r.DB.Get(&gate, `SELECT O_GATE FROM WEO_ORDER WHERE O_SEQ = ?`, orderSeq)
	return gate, err
}

func (r *DonateRepository) GetOrderPrice(orderSeq int) (int, string, error) {
	var row struct {
		Price   int    `db:"O_PRICE"`
		Payment string `db:"O_PAYMENT"`
	}
	err := r.DB.Get(&row, `SELECT O_PRICE, O_PAYMENT FROM WEO_ORDER WHERE O_SEQ = ?`, orderSeq)
	return row.Price, row.Payment, err
}
