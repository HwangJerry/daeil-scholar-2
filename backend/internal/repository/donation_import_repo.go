package repository

import (
	"fmt"
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

func (r *DonationImportRepository) FindMemberCandidatesByKeys(keys []model.DonationImportMemberKey) (map[model.DonationImportMemberKey][]model.MemberCandidate, error) {
	return findMemberCandidatesByKeys(r.DB, keys, false)
}

func (r *DonationImportRepository) FindMemberCandidatesByKeysTx(tx *sqlx.Tx, keys []model.DonationImportMemberKey) (map[model.DonationImportMemberKey][]model.MemberCandidate, error) {
	return findMemberCandidatesByKeys(tx, keys, true)
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

type donationImportMemberRow struct {
	MatchName   string `db:"MATCH_NAME"`
	MatchCohort string `db:"MATCH_COHORT"`
	MatchPhone  string `db:"MATCH_PHONE"`
	USRSeq      int    `db:"USR_SEQ"`
	Name        string `db:"USR_NAME"`
}

func findMemberCandidatesByKeys(queryer donationImportQueryer, keys []model.DonationImportMemberKey, lockForUpdate bool) (map[model.DonationImportMemberKey][]model.MemberCandidate, error) {
	result := make(map[model.DonationImportMemberKey][]model.MemberCandidate, len(keys))
	uniqueKeys := uniqueDonationImportMemberKeys(keys)
	if len(uniqueKeys) == 0 {
		return result, nil
	}

	requestRows := make([]string, 0, len(uniqueKeys))
	args := make([]interface{}, 0, len(uniqueKeys)*3)
	for index, key := range uniqueKeys {
		selectKeyword := "SELECT"
		if index > 0 {
			selectKeyword = "UNION ALL SELECT"
		}
		requestRows = append(requestRows, selectKeyword+` ? AS MATCH_NAME, ? AS MATCH_COHORT, ? AS MATCH_PHONE`)
		args = append(args, key.Name, key.Cohort, key.Phone)
	}
	query := `
		SELECT requested.MATCH_NAME, requested.MATCH_COHORT, requested.MATCH_PHONE,
		       member.USR_SEQ, member.USR_NAME
		FROM (` + strings.Join(requestRows, " ") + `) AS requested
		JOIN WEO_MEMBER AS member
		  ON member.USR_NAME = requested.MATCH_NAME
		 AND member.USR_FN = requested.MATCH_COHORT
		 AND (member.USR_PHONE = requested.MATCH_PHONE OR ` + legacyCanonicalPhoneSQL + ` = requested.MATCH_PHONE)
		WHERE member.USR_STATUS != 'AAA'
		ORDER BY member.USR_SEQ`
	if lockForUpdate {
		query += " FOR UPDATE"
	}
	rows := make([]donationImportMemberRow, 0)
	if err := queryer.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		key := model.NewDonationImportMemberKey(row.MatchName, row.MatchCohort, row.MatchPhone)
		result[key] = append(result[key], model.MemberCandidate{USRSeq: row.USRSeq, Name: row.Name})
	}
	return result, nil
}

func uniqueDonationImportMemberKeys(keys []model.DonationImportMemberKey) []model.DonationImportMemberKey {
	unique := make([]model.DonationImportMemberKey, 0, len(keys))
	seen := make(map[model.DonationImportMemberKey]struct{}, len(keys))
	for _, rawKey := range keys {
		key := model.NewDonationImportMemberKey(rawKey.Name, rawKey.Cohort, rawKey.Phone)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	return unique
}

func (r *DonationImportRepository) ExtRefExists(transactionNo, compositeKey string) (bool, error) {
	return donationExtRefExists(r.DB, transactionNo, compositeKey)
}

func (r *DonationImportRepository) ExtRefExistsTx(tx *sqlx.Tx, transactionNo, compositeKey string) (bool, error) {
	return donationExtRefExists(tx, transactionNo, compositeKey)
}

func (r *DonationImportRepository) FindExistingCompositeKeys(compositeKeys []string) (map[string]bool, error) {
	return findExistingDonationCompositeKeys(r.DB, compositeKeys)
}

func (r *DonationImportRepository) FindExistingCompositeKeysTx(tx *sqlx.Tx, compositeKeys []string) (map[string]bool, error) {
	return findExistingDonationCompositeKeys(tx, compositeKeys)
}

func findExistingDonationCompositeKeys(queryer donationImportQueryer, compositeKeys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(compositeKeys))
	uniqueKeys := make([]string, 0, len(compositeKeys))
	seen := make(map[string]struct{}, len(compositeKeys))
	for _, rawKey := range compositeKeys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result[key] = false
		uniqueKeys = append(uniqueKeys, key)
	}
	if len(uniqueKeys) == 0 {
		return result, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(uniqueKeys)), ",")
	args := make([]interface{}, len(uniqueKeys))
	for index, key := range uniqueKeys {
		args[index] = key
	}
	rows := make([]string, 0)
	query := fmt.Sprintf(`SELECT O_COMPOSITE_KEY FROM WEO_ORDER WHERE O_COMPOSITE_KEY IN (%s)`, placeholders)
	if err := queryer.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	for _, key := range rows {
		result[key] = true
	}
	return result, nil
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
