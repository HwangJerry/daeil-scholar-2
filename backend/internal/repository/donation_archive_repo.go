package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jmoiron/sqlx"
)

const DonationArchivePageSize = 50

type DonationArchiveMetadata struct {
	ID          int64  `db:"ID" json:"id"`
	Basis       string `db:"BASIS" json:"basis"`
	RetainUntil string `db:"RETAIN_UNTIL" json:"retainUntil"`
}
type DonationArchiveRepository struct{ DB *sqlx.DB }

func (r *DonationArchiveRepository) List(ctx context.Context, before int64) ([]DonationArchiveMetadata, error) {
	items := []DonationArchiveMetadata{}
	err := r.DB.SelectContext(ctx, &items, `SELECT ID,BASIS,DATE_FORMAT(RETAIN_UNTIL,'%Y-%m-%d') AS RETAIN_UNTIL
 FROM ALUMNI_DONATION_LEGAL_ARCHIVE WHERE RETAIN_UNTIL >= DATE(DATE_ADD(UTC_TIMESTAMP(),INTERVAL 9 HOUR))
 AND (?=0 OR ID<?) ORDER BY ID DESC LIMIT ?`, before, before, DonationArchivePageSize)
	return items, err
}

// Lock against expiry cleanup and commit the access receipt before releasing plaintext.
func (r *DonationArchiveRepository) Read(ctx context.Context, id int64, operator int, purpose string, open func([]byte) ([]byte, error)) (json.RawMessage, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var encrypted []byte
	err = tx.GetContext(ctx, &encrypted, `SELECT CIPHERTEXT FROM ALUMNI_DONATION_LEGAL_ARCHIVE
 WHERE ID=? AND RETAIN_UNTIL >= DATE(DATE_ADD(UTC_TIMESTAMP(),INTERVAL 9 HOUR)) FOR UPDATE`, id)
	if err != nil {
		return nil, err
	}
	plain, err := open(encrypted)
	if err != nil {
		return nil, err
	}
	if !json.Valid(plain) {
		return nil, errors.New("invalid archive payload")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ALUMNI_DONATION_ARCHIVE_ACCESS (ARCHIVE_ID,OPERATOR_SEQ,PURPOSE,ACCESSED_AT) VALUES (?,?,?,UTC_TIMESTAMP())`, id, operator, purpose)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return json.RawMessage(plain), nil
}
