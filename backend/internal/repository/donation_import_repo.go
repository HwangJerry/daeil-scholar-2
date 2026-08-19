package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func (r *DonationImportRepository) RunInTransaction(operation func(*sqlx.Tx) error) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := operation(tx); err != nil {
		return err
	}
	return tx.Commit()
}

type DonationImportRepository struct {
	DB *sqlx.DB
}

func NewDonationImportRepository(db *sqlx.DB) *DonationImportRepository {
	return &DonationImportRepository{DB: db}
}

func (r *DonationImportRepository) FindMemberCandidatesByNameCohortPhone(name, cohort, phone string) ([]model.MemberCandidate, error) {
	return findMemberCandidatesByNameCohortPhone(r.DB, name, cohort, phone, false)
}

func (r *DonationImportRepository) FindMemberCandidatesByNameCohortPhoneTx(tx *sqlx.Tx, name, cohort, phone string) ([]model.MemberCandidate, error) {
	return findMemberCandidatesByNameCohortPhone(tx, name, cohort, phone, true)
}

type donationImportQueryer interface {
	Select(dest interface{}, query string, args ...interface{}) error
	Get(dest interface{}, query string, args ...interface{}) error
}

func findMemberCandidatesByNameCohortPhone(queryer donationImportQueryer, name, cohort, phone string, lockForUpdate bool) ([]model.MemberCandidate, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone).String()
	candidates := make([]model.MemberCandidate, 0)
	query := `
		SELECT USR_SEQ, USR_NAME
		FROM WEO_MEMBER
		WHERE USR_NAME = ?
		  AND USR_FN = ?
		  AND (USR_PHONE = ? OR ` + legacyCanonicalPhoneSQL + ` = ?)
		  AND USR_STATUS != 'AAA'
		ORDER BY USR_SEQ
	`
	if lockForUpdate {
		query += " FOR UPDATE"
	}
	err := queryer.Select(&candidates, query, strings.TrimSpace(name), strings.TrimSpace(cohort), canonicalPhone, canonicalPhone)
	return candidates, err
}

func (r *DonationImportRepository) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	return donationExtRefExists(r.DB, transactionNo, compositeKey)
}

func (r *DonationImportRepository) ExtRefExistsTx(tx *sqlx.Tx, transactionNo, compositeKey string) (bool, error) {
	return donationExtRefExists(tx, transactionNo, compositeKey)
}

func donationExtRefExists(queryer donationImportQueryer, transactionNo, compositeKey string) (bool, error) {
	transactionNo = strings.TrimSpace(transactionNo)
	compositeKey = strings.TrimSpace(compositeKey)
	if transactionNo == "" && compositeKey == "" {
		return false, nil
	}

	var count int
	err := queryer.Get(&count, `
		SELECT COUNT(*)
		FROM WEO_ORDER
		WHERE (? != '' AND O_TRANSACTION_NO = ?)
		   OR (? != '' AND O_COMPOSITE_KEY = ?)
	`, transactionNo, transactionNo, compositeKey, compositeKey)
	return count > 0, err
}
