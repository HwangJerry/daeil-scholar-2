// account_erasure_donations.go — Archive only reviewed donation fields, then erase app copies.
package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"time"
)

type archiveDonation struct {
	ID          int    `db:"O_SEQ" json:"-"`
	Name        string `db:"DONOR_NAME" json:"donorName"`
	Date        string `db:"DONATION_DATE" json:"donationDate"`
	Gross       int64  `db:"O_GROSS_AMOUNT" json:"grossAmount"`
	Refund      int64  `db:"O_REFUNDED_AMOUNT" json:"refundedAmount"`
	Net         int64  `db:"O_NET_RECEIVED_AMOUNT" json:"netAmount"`
	Source      string `db:"O_SOURCE" json:"source"`
	Transaction string `db:"TRANSACTION_NO" json:"transactionNumber"`
	BasisDate   string `json:"basisDate"`
	Evidence    string `json:"evidenceReference"`
}

func eraseDonations(tx *sqlx.Tx, s erasureSchema, w model.ErasureWork, seal func([]byte) ([]byte, error), validate func(model.DonationRetentionDecision) error) error {
	if s["WEO_ORDER"] == nil {
		return nil
	}
	var rows []archiveDonation
	err := tx.Select(&rows, `SELECT O_SEQ,COALESCE(O_DONOR_NAME,m.USR_NAME,'') AS DONOR_NAME,
        COALESCE(DATE_FORMAT(O_DONATION_DATE,'%Y-%m-%d'),'') AS DONATION_DATE,
        COALESCE(O_GROSS_AMOUNT,0) AS O_GROSS_AMOUNT,O_REFUNDED_AMOUNT,COALESCE(O_NET_RECEIVED_AMOUNT,0) AS O_NET_RECEIVED_AMOUNT,
        O_SOURCE,COALESCE(O_TRANSACTION_NO,'') AS TRANSACTION_NO
        FROM WEO_ORDER o LEFT JOIN WEO_MEMBER m ON m.USR_SEQ=o.USR_SEQ
        WHERE o.USR_SEQ=? OR o.O_ACCOUNT_USR_SEQ=? FOR UPDATE`, w.UserSeq, w.UserSeq)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	// Serialize the aggregate transfer, including concurrent erasures.
	var lock int
	if err = tx.Get(&lock, `SELECT ID FROM ALUMNI_ERASED_DONATION_TOTAL WHERE ID=1 FOR UPDATE`); err != nil {
		return err
	}
	beforeAmount, beforeCount, err := getReceivedDonationAggregate(tx)
	if err != nil {
		return err
	}
	var lastUntil *time.Time
	summaries := map[string]string{}
	summaryDates := map[string]string{}
	for _, row := range rows {
		var decision model.DonationRetentionDecision
		err = tx.Get(&decision, `SELECT O_SEQ,BASIS,BASIS_DATE,RETAIN_UNTIL,EVIDENCE_REFERENCE FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=?`, row.ID)
		if err == sql.ErrNoRows {
			return &model.ErasureBlocked{Code: "DONATION_RETENTION_REVIEW_REQUIRED"}
		}
		if err != nil {
			return err
		}
		if err = validate(decision); err != nil {
			return &model.ErasureBlocked{Code: "INVALID_DONATION_RETENTION_DECISION"}
		}
		if decision.Until != nil {
			// Expiry is inclusive in Korean local calendar time.
			korea, _ := time.LoadLocation("Asia/Seoul")
			end := time.Date(decision.Until.Year(), decision.Until.Month(), decision.Until.Day()+1, 0, 0, 0, 0, korea)
			if end.After(time.Now()) {
				row.BasisDate = decision.BasisDate.Format("2006-01-02")
				row.Evidence = decision.Evidence
				plain, e := json.Marshal(row)
				if e != nil {
					return e
				}
				if seal == nil {
					return &model.ErasureBlocked{Code: "DONATION_ARCHIVE_KEY_REQUIRED"}
				}
				encrypted, e := seal(plain)
				if e != nil {
					return e
				}
				if _, e = tx.Exec(`INSERT INTO ALUMNI_DONATION_LEGAL_ARCHIVE (BASIS,RETAIN_UNTIL,CIPHERTEXT) VALUES (?,?,?)`, decision.Basis, decision.Until.Format("2006-01-02"), encrypted); e != nil {
					return e
				}
				if lastUntil == nil || decision.Until.After(*lastUntil) {
					lastUntil = decision.Until
				}
				label := "기부 장부 최소 증빙(상속세 및 증여세법 제51조)"
				if decision.Basis == "receipt_5y" {
					label = "기부금영수증 관련 최소 증빙(소득세법 제160조의3/법인세법 제112조의2 중 해당 근거)"
				}
				if day := decision.Until.Format("2006-01-02"); day > summaryDates[decision.Basis] {
					summaries[decision.Basis] = label + ": " + day + "까지"
					summaryDates[decision.Basis] = day
				}
			}
		}
		if err = s.erase(tx, "WEO_PG_DATA", "O_SEQ=?", row.ID); err != nil {
			return err
		}
		if err = s.erase(tx, "WEO_ORDER", "O_SEQ=?", row.ID); err != nil {
			return err
		}
		if _, err = tx.Exec(`DELETE FROM ALUMNI_DONATION_RETENTION WHERE O_SEQ=?`, row.ID); err != nil {
			return err
		}
	}
	afterAmount, afterCount, err := getReceivedDonationAggregate(tx)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE ALUMNI_ERASED_DONATION_TOTAL SET TOTAL_AMOUNT=TOTAL_AMOUNT+?,DONOR_COUNT=DONOR_COUNT+? WHERE ID=1`, beforeAmount-afterAmount, beforeCount-afterCount); err != nil {
		return err
	}
	summary := "없음"
	var until interface{}
	if lastUntil != nil {
		summary = "기부자명·기부일·금액·거래 증빙만 암호화 분리 보관. "
		for _, basis := range []string{"ledger_10y", "receipt_5y"} {
			if v := summaries[basis]; v != "" {
				summary += v + ". "
			}
		}
		until = lastUntil.Format("2006-01-02")
	}
	_, err = tx.Exec(`UPDATE ALUMNI_ACCOUNT_DELETION_REQUEST SET RETAINED_RECORDS=?,RETENTION_UNTIL=? WHERE REQUEST_ID=?`, summary, until, w.RequestID)
	return err
}

// Reviewed dates must be supplied explicitly; no guess based on today's date.
func (r *AccountDeletionRequestRepository) SaveDonationRetention(d model.DonationRetentionDecision) error {
	var date, until interface{}
	if d.BasisDate != nil {
		date = d.BasisDate.Format("2006-01-02")
	}
	if d.Until != nil {
		until = d.Until.Format("2006-01-02")
	}
	result, err := r.DB.Exec(`INSERT INTO ALUMNI_DONATION_RETENTION (O_SEQ,BASIS,BASIS_DATE,RETAIN_UNTIL,EVIDENCE_REFERENCE,REVIEWED_AT)
        SELECT O_SEQ,?,?,?,?,UTC_TIMESTAMP() FROM WEO_ORDER WHERE O_SEQ=?
        ON DUPLICATE KEY UPDATE BASIS=VALUES(BASIS),BASIS_DATE=VALUES(BASIS_DATE),RETAIN_UNTIL=VALUES(RETAIN_UNTIL),EVIDENCE_REFERENCE=VALUES(EVIDENCE_REFERENCE),REVIEWED_AT=UTC_TIMESTAMP()`, d.Basis, date, until, d.Evidence, d.OrderID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("donation not found or unchanged")
	}
	return nil
}
