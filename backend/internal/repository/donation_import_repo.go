package repository

import (
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type DonationImportRepository struct {
	DB *sqlx.DB
}

func NewDonationImportRepository(db *sqlx.DB) *DonationImportRepository {
	return &DonationImportRepository{DB: db}
}

func (r *DonationImportRepository) FindMemberCandidatesByNamePhone(name, phone string) ([]model.MemberCandidate, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone).String()
	candidates := make([]model.MemberCandidate, 0)
	err := r.DB.Select(&candidates, `
		SELECT USR_SEQ, USR_NAME
		FROM WEO_MEMBER
		WHERE USR_NAME = ?
		  AND (USR_PHONE = ? OR `+legacyCanonicalPhoneSQL+` = ?)
		  AND USR_STATUS != 'AAA'
		ORDER BY USR_SEQ
	`, strings.TrimSpace(name), canonicalPhone, canonicalPhone)
	return candidates, err
}

func (r *DonationImportRepository) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	transactionNo = strings.TrimSpace(transactionNo)
	compositeKey = strings.TrimSpace(compositeKey)
	if transactionNo == "" && compositeKey == "" {
		return false, nil
	}

	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*)
		FROM WEO_ORDER
		WHERE (? != '' AND O_TRANSACTION_NO = ?)
		   OR (? != '' AND O_COMPOSITE_KEY = ?)
	`, transactionNo, transactionNo, compositeKey, compositeKey)
	return count > 0, err
}
