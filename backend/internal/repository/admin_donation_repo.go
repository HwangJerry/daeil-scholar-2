// Admin donation repository — config update and snapshot history queries
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	ErrDonationOrderConflict   = errors.New("donation order identity conflict")
	ErrDonationOrderStale      = errors.New("donation order was edited concurrently")
	ErrDonationOrderNotFound   = errors.New("donation order not found")
	ErrDonationAccountNotFound = errors.New("donation account not found")
)

type donationOrderRow struct {
	OrderSeq          int64          `db:"ORDER_SEQ"`
	AccountUsrSeq     sql.NullInt64  `db:"ACCOUNT_USR_SEQ"`
	Source            string         `db:"SOURCE"`
	TransactionNumber sql.NullString `db:"TRANSACTION_NUMBER"`
	DonationDate      string         `db:"DONATION_DATE"`
	DonorName         string         `db:"DONOR_NAME"`
	DonorCohort       string         `db:"DONOR_COHORT"`
	DonorDepartment   string         `db:"DONOR_DEPARTMENT"`
	DonorPhone        string         `db:"DONOR_PHONE"`
	DonationType      string         `db:"DONATION_TYPE"`
	GrossAmount       int64          `db:"GROSS_AMOUNT"`
	RefundedAmount    int64          `db:"REFUNDED_AMOUNT"`
	NetReceivedAmount int64          `db:"NET_RECEIVED_AMOUNT"`
	Status            string         `db:"STATUS"`
	PaymentMethod     string         `db:"PAYMENT_METHOD"`
	Memo              sql.NullString `db:"MEMO"`
	LastEditedBy      int            `db:"LAST_EDITED_BY"`
	LastEditedAt      string         `db:"LAST_EDITED_AT"`
	LastEditedIP      string         `db:"LAST_EDITED_IP"`
}

type AdminDonationRepository struct {
	DB *sqlx.DB
}

func NewAdminDonationRepository(db *sqlx.DB) *AdminDonationRepository {
	return &AdminDonationRepository{DB: db}
}

func (r *AdminDonationRepository) CreateDonationOrder(order model.NormalizedDonationOrder, operSeq int, ip string) (int64, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	seq, err := r.CreateDonationOrderTx(tx, order, operSeq, ip)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return seq, nil
}

// CreateDonationOrderTx persists a canonical donation using the caller's
// transaction. Batch import uses this to keep every approved row atomic.
func (r *AdminDonationRepository) CreateDonationOrderTx(tx *sqlx.Tx, order model.NormalizedDonationOrder, operSeq int, ip string) (int64, error) {
	if err := lockActiveDonationAccount(tx, order.AccountUsrSeq); err != nil {
		return 0, err
	}

	result, err := tx.Exec(`
		INSERT INTO WEO_ORDER (
			USR_SEQ, O_ACCOUNT_USR_SEQ, O_SOURCE, O_TRANSACTION_NO, O_COMPOSITE_KEY, O_DONATION_DATE,
			O_DONOR_NAME, O_DONOR_PHONE, O_DONOR_COHORT, O_DONOR_DEPARTMENT, O_GATE,
			O_GROSS_AMOUNT, O_REFUNDED_AMOUNT, O_NET_RECEIVED_AMOUNT, O_LIFECYCLE_STATUS,
			O_PAYMENT_METHOD, O_MEMO, O_PRICE, O_PAY, O_PAY_TYPE, O_STATUS, O_PAYMENT,
			O_TYPE, O_REGDATE, REG_OPER, REG_DATE, REG_IPADDR, EDT_OPER, EDT_DATE, EDT_IPADDR
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			'A', NOW(), ?, NOW(), ?, ?, NOW(), ?
		)
	`,
		accountUsrSeqOrZero(order.AccountUsrSeq), order.AccountUsrSeq, order.Source, order.TransactionNumber, nullableCompositeKey(order), order.DonationDate,
		order.Donor.Name, order.Donor.Phone, order.Donor.Cohort, order.Donor.Department, order.LegacyGate,
		order.GrossAmount, order.RefundedAmount, order.NetReceivedAmount, order.Status,
		order.PaymentMethod, order.Memo, order.GrossAmount, order.NetReceivedAmount,
		legacyDonationPayType(order.PaymentMethod), order.LegacyStatus, order.LegacyPayment,
		operSeq, ip, operSeq, ip,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, ErrDonationOrderConflict
		}
		return 0, err
	}
	seq, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (r *AdminDonationRepository) CreateDonationOrdersTx(tx *sqlx.Tx, orders []model.NormalizedDonationOrder, operSeq int, ip string) ([]int64, error) {
	if len(orders) == 0 {
		return []int64{}, nil
	}
	if err := lockActiveDonationAccounts(tx, orders); err != nil {
		return nil, err
	}

	const valueClause = `(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'A', NOW(), ?, NOW(), ?, ?, NOW(), ?)`
	values := make([]string, len(orders))
	args := make([]interface{}, 0, len(orders)*26)
	for index, order := range orders {
		values[index] = valueClause
		args = append(args,
			accountUsrSeqOrZero(order.AccountUsrSeq), order.AccountUsrSeq, order.Source, order.TransactionNumber, nullableCompositeKey(order), order.DonationDate,
			order.Donor.Name, order.Donor.Phone, order.Donor.Cohort, order.Donor.Department, order.LegacyGate,
			order.GrossAmount, order.RefundedAmount, order.NetReceivedAmount, order.Status,
			order.PaymentMethod, order.Memo, order.GrossAmount, order.NetReceivedAmount,
			legacyDonationPayType(order.PaymentMethod), order.LegacyStatus, order.LegacyPayment,
			operSeq, ip, operSeq, ip,
		)
	}
	query := `
		INSERT INTO WEO_ORDER (
			USR_SEQ, O_ACCOUNT_USR_SEQ, O_SOURCE, O_TRANSACTION_NO, O_COMPOSITE_KEY, O_DONATION_DATE,
			O_DONOR_NAME, O_DONOR_PHONE, O_DONOR_COHORT, O_DONOR_DEPARTMENT, O_GATE,
			O_GROSS_AMOUNT, O_REFUNDED_AMOUNT, O_NET_RECEIVED_AMOUNT, O_LIFECYCLE_STATUS,
			O_PAYMENT_METHOD, O_MEMO, O_PRICE, O_PAY, O_PAY_TYPE, O_STATUS, O_PAYMENT,
			O_TYPE, O_REGDATE, REG_OPER, REG_DATE, REG_IPADDR, EDT_OPER, EDT_DATE, EDT_IPADDR
		) VALUES ` + strings.Join(values, ", ")
	if _, err := tx.Exec(query, args...); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDonationOrderConflict
		}
		return nil, err
	}

	return findInsertedDonationOrderSequences(tx, orders)
}

func lockActiveDonationAccounts(tx *sqlx.Tx, orders []model.NormalizedDonationOrder) error {
	unique := make(map[int]struct{})
	for _, order := range orders {
		if order.AccountUsrSeq != nil {
			unique[*order.AccountUsrSeq] = struct{}{}
		}
	}
	accountUsrSeqs := make([]int, 0, len(unique))
	for usrSeq := range unique {
		accountUsrSeqs = append(accountUsrSeqs, usrSeq)
	}
	if len(accountUsrSeqs) == 0 {
		return nil
	}
	sort.Ints(accountUsrSeqs)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(accountUsrSeqs)), ",")
	args := make([]interface{}, len(accountUsrSeqs))
	for index, usrSeq := range accountUsrSeqs {
		args[index] = usrSeq
	}
	locked := make([]int, 0, len(accountUsrSeqs))
	query := fmt.Sprintf(`
		SELECT USR_SEQ
		FROM WEO_MEMBER
		WHERE USR_SEQ IN (%s) AND USR_STATUS != 'AAA'
		ORDER BY USR_SEQ
		FOR UPDATE`, placeholders)
	if err := tx.Select(&locked, query, args...); err != nil {
		return err
	}
	if len(locked) != len(accountUsrSeqs) {
		return ErrDonationAccountNotFound
	}
	return nil
}

type insertedDonationOrderRow struct {
	OrderSeq     int64  `db:"O_SEQ"`
	CompositeKey string `db:"O_COMPOSITE_KEY"`
}

func findInsertedDonationOrderSequences(tx *sqlx.Tx, orders []model.NormalizedDonationOrder) ([]int64, error) {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(orders)), ",")
	args := make([]interface{}, len(orders))
	for index, order := range orders {
		args[index] = order.CompositeKey
	}
	rows := make([]insertedDonationOrderRow, 0, len(orders))
	query := fmt.Sprintf(`SELECT O_SEQ, O_COMPOSITE_KEY FROM WEO_ORDER WHERE O_COMPOSITE_KEY IN (%s)`, placeholders)
	if err := tx.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	sequenceByKey := make(map[string]int64, len(rows))
	for _, row := range rows {
		sequenceByKey[row.CompositeKey] = row.OrderSeq
	}
	sequences := make([]int64, len(orders))
	for index, order := range orders {
		sequence, exists := sequenceByKey[order.CompositeKey]
		if !exists {
			return nil, errors.New("inserted donation order not found")
		}
		sequences[index] = sequence
	}
	return sequences, nil
}

// lockActiveDonationAccount serializes donation linking with account deletion,
// which locks the same member row before unlinking donations and anonymizing it.
func lockActiveDonationAccount(tx *sqlx.Tx, accountUsrSeq *int) error {
	if accountUsrSeq == nil {
		return nil
	}
	var lockedUsrSeq int
	err := tx.Get(&lockedUsrSeq, `
		SELECT USR_SEQ
		FROM WEO_MEMBER
		WHERE USR_SEQ = ? AND USR_STATUS != 'AAA'
		FOR UPDATE
	`, *accountUsrSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDonationAccountNotFound
	}
	return err
}

func (r *AdminDonationRepository) GetDonationOrder(seq int64) (*model.DonationOrder, error) {
	var row donationOrderRow
	err := r.DB.Get(&row, `
		SELECT
			o.O_SEQ AS ORDER_SEQ,
			o.O_ACCOUNT_USR_SEQ AS ACCOUNT_USR_SEQ,
			COALESCE(o.O_SOURCE, 'other') AS SOURCE,
			o.O_TRANSACTION_NO AS TRANSACTION_NUMBER,
			DATE_FORMAT(o.O_DONATION_DATE, '%Y-%m-%d') AS DONATION_DATE,
			COALESCE(o.O_DONOR_NAME, m.USR_NAME, '') AS DONOR_NAME,
			COALESCE(o.O_DONOR_COHORT, m.USR_FN, '') AS DONOR_COHORT,
			COALESCE(o.O_DONOR_DEPARTMENT, m.USR_DEPT, '') AS DONOR_DEPARTMENT,
			COALESCE(o.O_DONOR_PHONE, m.USR_PHONE, '') AS DONOR_PHONE,
			CASE o.O_GATE WHEN 'P' THEN 'recurring' WHEN 'S' THEN 'one_time' WHEN 'F' THEN 'sponsorship' ELSE '' END AS DONATION_TYPE,
			COALESCE(o.O_GROSS_AMOUNT, o.O_PRICE, 0) AS GROSS_AMOUNT,
			COALESCE(o.O_REFUNDED_AMOUNT, 0) AS REFUNDED_AMOUNT,
			COALESCE(o.O_NET_RECEIVED_AMOUNT, 0) AS NET_RECEIVED_AMOUNT,
			COALESCE(o.O_LIFECYCLE_STATUS, 'pending') AS STATUS,
			COALESCE(o.O_PAYMENT_METHOD, 'other') AS PAYMENT_METHOD,
			o.O_MEMO AS MEMO,
			COALESCE(o.EDT_OPER, o.REG_OPER, 0) AS LAST_EDITED_BY,
			COALESCE(DATE_FORMAT(o.EDT_DATE, '%Y-%m-%dT%H:%i:%sZ'), DATE_FORMAT(o.REG_DATE, '%Y-%m-%dT%H:%i:%sZ'), '') AS LAST_EDITED_AT,
			COALESCE(o.EDT_IPADDR, o.REG_IPADDR, '') AS LAST_EDITED_IP
		FROM WEO_ORDER o
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = o.USR_SEQ
		WHERE o.O_SEQ = ? AND o.O_TYPE = 'A'
		LIMIT 1
	`, seq)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDonationOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return donationOrderFromRow(row), nil
}

func (r *AdminDonationRepository) ListDonationOrders(filters model.DonationOrderFilters, page, size int) ([]*model.DonationOrder, int, error) {
	conditions := []string{"o.O_TYPE = 'A'"}
	args := []interface{}{}
	if filters.Name != "" {
		conditions = append(conditions, "COALESCE(o.O_DONOR_NAME, m.USR_NAME, '') LIKE ?")
		args = append(args, "%"+filters.Name+"%")
	}
	if filters.Phone != "" {
		conditions = append(conditions, "COALESCE(o.O_DONOR_PHONE, m.USR_PHONE, '') LIKE ?")
		args = append(args, "%"+filters.Phone+"%")
	}
	if filters.TransactionNumber != "" {
		conditions = append(conditions, "o.O_TRANSACTION_NO LIKE ?")
		args = append(args, "%"+filters.TransactionNumber+"%")
	}
	if filters.Source != "" {
		conditions = append(conditions, "o.O_SOURCE = ?")
		args = append(args, filters.Source)
	}
	if filters.Status != "" {
		conditions = append(conditions, "o.O_LIFECYCLE_STATUS = ?")
		args = append(args, filters.Status)
	}
	if filters.DonationType != "" {
		conditions = append(conditions, "o.O_GATE = ?")
		args = append(args, legacyDonationGate(filters.DonationType))
	}
	where := " WHERE " + strings.Join(conditions, " AND ")
	var total int
	if err := r.DB.Get(&total, `SELECT COUNT(*) FROM WEO_ORDER o LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = o.USR_SEQ`+where, args...); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]interface{}{}, args...), size, (page-1)*size)
	var rows []donationOrderRow
	err := r.DB.Select(&rows, `
		SELECT
			o.O_SEQ AS ORDER_SEQ, o.O_ACCOUNT_USR_SEQ AS ACCOUNT_USR_SEQ,
			COALESCE(o.O_SOURCE, 'other') AS SOURCE,
			o.O_TRANSACTION_NO AS TRANSACTION_NUMBER, DATE_FORMAT(o.O_DONATION_DATE, '%Y-%m-%d') AS DONATION_DATE,
			COALESCE(o.O_DONOR_NAME, m.USR_NAME, '') AS DONOR_NAME,
			COALESCE(o.O_DONOR_COHORT, m.USR_FN, '') AS DONOR_COHORT,
			COALESCE(o.O_DONOR_DEPARTMENT, m.USR_DEPT, '') AS DONOR_DEPARTMENT,
			COALESCE(o.O_DONOR_PHONE, m.USR_PHONE, '') AS DONOR_PHONE,
			CASE o.O_GATE WHEN 'P' THEN 'recurring' WHEN 'S' THEN 'one_time' WHEN 'F' THEN 'sponsorship' ELSE '' END AS DONATION_TYPE,
			COALESCE(o.O_GROSS_AMOUNT, o.O_PRICE, 0) AS GROSS_AMOUNT,
			COALESCE(o.O_REFUNDED_AMOUNT, 0) AS REFUNDED_AMOUNT,
			COALESCE(o.O_NET_RECEIVED_AMOUNT, 0) AS NET_RECEIVED_AMOUNT,
			COALESCE(o.O_LIFECYCLE_STATUS, 'pending') AS STATUS,
			COALESCE(o.O_PAYMENT_METHOD, 'other') AS PAYMENT_METHOD,
			o.O_MEMO AS MEMO, COALESCE(o.EDT_OPER, o.REG_OPER, 0) AS LAST_EDITED_BY,
			COALESCE(DATE_FORMAT(o.EDT_DATE, '%Y-%m-%dT%H:%i:%sZ'), DATE_FORMAT(o.REG_DATE, '%Y-%m-%dT%H:%i:%sZ'), '') AS LAST_EDITED_AT,
			COALESCE(o.EDT_IPADDR, o.REG_IPADDR, '') AS LAST_EDITED_IP
		FROM WEO_ORDER o
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = o.USR_SEQ`+where+`
		ORDER BY o.O_DONATION_DATE DESC, o.O_SEQ DESC
		LIMIT ? OFFSET ?
	`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	orders := make([]*model.DonationOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, donationOrderFromRow(row))
	}
	return orders, total, nil
}

func donationOrderFromRow(row donationOrderRow) *model.DonationOrder {
	var accountUsrSeq *int
	if row.AccountUsrSeq.Valid {
		value := int(row.AccountUsrSeq.Int64)
		accountUsrSeq = &value
	}
	var transactionNumber, memo *string
	if row.TransactionNumber.Valid {
		value := row.TransactionNumber.String
		transactionNumber = &value
	}
	if row.Memo.Valid {
		value := row.Memo.String
		memo = &value
	}
	return &model.DonationOrder{
		OrderSeq: row.OrderSeq, AccountUsrSeq: accountUsrSeq, Source: row.Source, TransactionNumber: transactionNumber,
		DonationDate: row.DonationDate,
		Donor:        model.DonationDonor{Name: row.DonorName, Cohort: row.DonorCohort, Department: row.DonorDepartment, Phone: row.DonorPhone},
		DonationType: row.DonationType, GrossAmount: row.GrossAmount, RefundedAmount: row.RefundedAmount,
		NetReceivedAmount: row.NetReceivedAmount, Status: row.Status, PaymentMethod: row.PaymentMethod,
		Memo: memo, LastEditedBy: row.LastEditedBy, LastEditedAt: row.LastEditedAt, LastEditedIP: row.LastEditedIP,
	}
}

func nullableCompositeKey(order model.NormalizedDonationOrder) interface{} {
	if order.TransactionNumber != nil {
		return nil
	}
	return order.CompositeKey
}

func accountUsrSeqOrZero(accountUsrSeq *int) int {
	if accountUsrSeq == nil {
		return 0
	}
	return *accountUsrSeq
}

func legacyDonationPayType(paymentMethod string) string {
	switch paymentMethod {
	case "card":
		return "CARD"
	case "bank":
		return "BANK"
	case "virtual_bank":
		return "VBANK"
	case "mobile":
		return "HP"
	case "admin":
		return "ADMS"
	default:
		return "FREE"
	}
}

func legacyDonationGate(donationType string) string {
	switch donationType {
	case "recurring":
		return "P"
	case "one_time":
		return "S"
	case "sponsorship":
		return "F"
	default:
		return ""
	}
}

func (r *AdminDonationRepository) UpdateConfig(config model.DonationConfig, operSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE DONATION_CONFIG
		SET DC_GOAL = ?, DC_MANUAL_ADJ = ?, DC_MANUAL_DONOR_CNT = ?,
		    DC_TIER_SPROUT_MIN = ?, DC_TIER_SAPLING_MIN = ?, DC_TIER_TREE_MIN = ?,
		    DC_TIER_BLOOMING_MIN = ?, DC_TIER_FRUITING_MIN = ?,
		    DC_NOTE = ?, DC_OVERWRITE = ?, REG_OPER = ?, REG_DATE = NOW()
		WHERE IS_ACTIVE = 'Y'
	`, config.Goal, config.ManualAdj, config.ManualDonorCnt,
		config.TierSproutMin, config.TierSaplingMin, config.TierTreeMin,
		config.TierBloomingMin, config.TierFruitingMin,
		config.Note, config.Overwrite, operSeq)
	return err
}

func (r *AdminDonationRepository) GetSnapshotHistory(days int) ([]model.DonationSnapshot, error) {
	var snapshots []model.DonationSnapshot
	err := r.DB.Select(&snapshots, `
		SELECT DS_SEQ, DS_DATE, DS_TOTAL, DS_MANUAL_ADJ, DS_DONOR_CNT, DS_GOAL,
		       IFNULL(DS_OVERWRITE,'N') AS DS_OVERWRITE,
		       IFNULL(DATE_FORMAT(REG_DATE,'%Y-%m-%d %H:%i:%s'),'') AS REG_DATE
		FROM DONATION_SNAPSHOT
		ORDER BY DS_DATE DESC
		LIMIT ?
	`, days)
	return snapshots, err
}

func (r *AdminDonationRepository) GetDonationOrders(page, size int, search, status, payType string) ([]model.AdminDonationOrderRow, int, error) {
	args := []interface{}{}
	conditions := []string{"1=1"}
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

func (r *AdminDonationRepository) UpdateDonationOrder(seq int64, order model.NormalizedDonationOrder, operSeq int, ip string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if order.AccountUsrSeqSet {
		if err := lockActiveDonationAccount(tx, order.AccountUsrSeq); err != nil {
			return err
		}
	}

	result, err := tx.Exec(`
		UPDATE WEO_ORDER SET
			O_ACCOUNT_USR_SEQ = CASE WHEN ? THEN ? ELSE O_ACCOUNT_USR_SEQ END,
			USR_SEQ = CASE WHEN ? THEN COALESCE(?, 0) ELSE USR_SEQ END,
			O_ACCOUNT_UNLINKED_AT = CASE
				WHEN NOT ? THEN O_ACCOUNT_UNLINKED_AT
				WHEN ? IS NULL THEN NOW()
				ELSE NULL
			END,
			O_SOURCE = ?, O_TRANSACTION_NO = ?, O_COMPOSITE_KEY = ?, O_DONATION_DATE = ?,
			O_DONOR_NAME = ?, O_DONOR_PHONE = ?, O_DONOR_COHORT = ?, O_DONOR_DEPARTMENT = ?, O_GATE = ?,
			O_GROSS_AMOUNT = ?, O_REFUNDED_AMOUNT = ?, O_NET_RECEIVED_AMOUNT = ?, O_LIFECYCLE_STATUS = ?,
			O_PAYMENT_METHOD = ?, O_MEMO = ?, O_PRICE = ?, O_PAY = ?, O_PAY_TYPE = ?, O_STATUS = ?, O_PAYMENT = ?,
			EDT_OPER = ?, EDT_DATE = NOW(), EDT_IPADDR = ?
		WHERE O_SEQ = ? AND O_TYPE = 'A'
		  AND COALESCE(EDT_DATE, REG_DATE) = STR_TO_DATE(?, '%Y-%m-%dT%H:%i:%sZ')
	`,
		order.AccountUsrSeqSet, order.AccountUsrSeq,
		order.AccountUsrSeqSet, order.AccountUsrSeq,
		order.AccountUsrSeqSet, order.AccountUsrSeq,
		order.Source, order.TransactionNumber, nullableCompositeKey(order), order.DonationDate,
		order.Donor.Name, order.Donor.Phone, order.Donor.Cohort, order.Donor.Department, order.LegacyGate,
		order.GrossAmount, order.RefundedAmount, order.NetReceivedAmount, order.Status,
		order.PaymentMethod, order.Memo, order.GrossAmount, order.NetReceivedAmount,
		legacyDonationPayType(order.PaymentMethod), order.LegacyStatus, order.LegacyPayment,
		operSeq, ip, seq, order.LastEditedAt,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrDonationOrderConflict
		}
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var currentLastEditedAt string
		err := tx.Get(&currentLastEditedAt, `
			SELECT COALESCE(
				DATE_FORMAT(EDT_DATE, '%Y-%m-%dT%H:%i:%sZ'),
				DATE_FORMAT(REG_DATE, '%Y-%m-%dT%H:%i:%sZ'),
				''
			)
			FROM WEO_ORDER
			WHERE O_SEQ = ? AND O_TYPE = 'A'
			FOR UPDATE
		`, seq)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDonationOrderNotFound
		}
		if err != nil {
			return err
		}
		if currentLastEditedAt != order.LastEditedAt {
			return ErrDonationOrderStale
		}
	}
	return tx.Commit()
}
