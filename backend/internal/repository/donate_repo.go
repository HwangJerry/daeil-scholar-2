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

// InsertOrder populates both canonical donation ledger fields and the legacy
// EasyPay compatibility fields. The corresponding routes are currently disabled,
// but keeping this writer canonical prevents summary drift when they are restored.
func (r *DonateRepository) InsertOrder(usrSeq int, gate string, payType string, price int, ip string) (int64, error) {
	result, err := r.DB.Exec(`
		INSERT INTO WEO_ORDER (
			USR_SEQ, O_ACCOUNT_USR_SEQ, O_SOURCE, O_DONATION_DATE,
			O_GATE, O_GROSS_AMOUNT, O_REFUNDED_AMOUNT, O_NET_RECEIVED_AMOUNT,
			O_LIFECYCLE_STATUS, O_PAYMENT_METHOD,
			O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR
		)
		VALUES (?, ?, 'other', CURRENT_DATE, ?, ?, 0, 0, 'pending', ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
	`, usrSeq, usrSeq, gate, price, canonicalPaymentMethod(payType), payType, price, ip)
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

// InsertOrderTx creates the same canonical pending order within an existing tx.
func (r *DonateRepository) InsertOrderTx(tx *sqlx.Tx, usrSeq int, gate, payType string, price int, ip string) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO WEO_ORDER (
			USR_SEQ, O_ACCOUNT_USR_SEQ, O_SOURCE, O_DONATION_DATE,
			O_GATE, O_GROSS_AMOUNT, O_REFUNDED_AMOUNT, O_NET_RECEIVED_AMOUNT,
			O_LIFECYCLE_STATUS, O_PAYMENT_METHOD,
			O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR
		)
		VALUES (?, ?, 'other', CURRENT_DATE, ?, ?, 0, 0, 'pending', ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
	`, usrSeq, usrSeq, gate, price, canonicalPaymentMethod(payType), payType, price, ip)
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

func (r *DonateRepository) UpdateOrderPayment(orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
	result, err := r.DB.Exec(`
		UPDATE WEO_ORDER
		SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
		    O_NET_RECEIVED_AMOUNT = ?, O_LIFECYCLE_STATUS = 'completed',
		    EDT_DATE = NOW(), EDT_IPADDR = ?
		WHERE O_SEQ = ? AND O_PAYMENT = 'N'
	`, amount, pgSeq, amount, ip, orderSeq)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// UpdateOrderPaymentTx marks both canonical and legacy payment state as completed.
func (r *DonateRepository) UpdateOrderPaymentTx(tx *sqlx.Tx, orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
	result, err := tx.Exec(`
		UPDATE WEO_ORDER
		SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
		    O_NET_RECEIVED_AMOUNT = ?, O_LIFECYCLE_STATUS = 'completed',
		    EDT_DATE = NOW(), EDT_IPADDR = ?
		WHERE O_SEQ = ? AND O_PAYMENT = 'N'
	`, amount, pgSeq, amount, ip, orderSeq)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func canonicalPaymentMethod(payType string) string {
	switch payType {
	case "CARD":
		return "card"
	case "BANK":
		return "bank"
	case "VBANK":
		return "virtual_bank"
	case "HP":
		return "mobile"
	default:
		return "other"
	}
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
