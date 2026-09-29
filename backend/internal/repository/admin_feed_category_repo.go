// admin_feed_category_repo.go — CRUD, reorder and post-moving delete queries for ALUMNI_FEED_CATEGORY
package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminFeedCategoryRepository struct {
	DB *sqlx.DB
}

func NewAdminFeedCategoryRepository(db *sqlx.DB) *AdminFeedCategoryRepository {
	return &AdminFeedCategoryRepository{DB: db}
}

// GetAll returns every category (open and hidden) in SORT_ORDER with its NOTICE
// post count. Posts with a NULL or dangling category count toward the default.
func (r *AdminFeedCategoryRepository) GetAll() ([]model.AdminFeedCategory, error) {
	cats := make([]model.AdminFeedCategory, 0)
	err := r.DB.Select(&cats, `
		SELECT c.FC_SEQ, c.FC_CODE, c.FC_NAME, c.SORT_ORDER, c.OPEN_YN, c.IS_DEFAULT,
		       IFNULL(p.post_count, 0) AS post_count
		FROM ALUMNI_FEED_CATEGORY c
		LEFT JOIN (
			SELECT `+feedCategorySeqExpr+` AS category_seq, COUNT(*) AS post_count
			FROM WEO_BOARDBBS b`+feedCategoryJoin+`
			WHERE b.GATE = 'NOTICE'
			GROUP BY category_seq
		) p ON p.category_seq = c.FC_SEQ
		ORDER BY c.SORT_ORDER ASC, c.FC_SEQ ASC
	`)
	return cats, err
}

// GetBySeq returns one category without its post count, or nil when absent.
func (r *AdminFeedCategoryRepository) GetBySeq(seq int) (*model.AdminFeedCategory, error) {
	var cat model.AdminFeedCategory
	err := r.DB.Get(&cat, `
		SELECT FC_SEQ, FC_CODE, FC_NAME, SORT_ORDER, OPEN_YN, IS_DEFAULT, 0 AS post_count
		FROM ALUMNI_FEED_CATEGORY WHERE FC_SEQ = ?
	`, seq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

// Insert appends a non-default category after the last one and gives it the
// stable code c<seq>. The row is first written under a random placeholder code
// because FC_CODE is NOT NULL UNIQUE and the seq is only known after the insert.
func (r *AdminFeedCategoryRepository) Insert(name, openYN string) (int, error) {
	placeholder, err := placeholderFeedCategoryCode()
	if err != nil {
		return 0, err
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`
		INSERT INTO ALUMNI_FEED_CATEGORY (FC_CODE, FC_NAME, SORT_ORDER, OPEN_YN, IS_DEFAULT, REG_DATE, UPD_DATE)
		SELECT ?, ?, IFNULL(MAX(SORT_ORDER), 0) + 1, ?, 'N', NOW(), NOW() FROM ALUMNI_FEED_CATEGORY
	`, placeholder, name, openYN)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`UPDATE ALUMNI_FEED_CATEGORY SET FC_CODE = CONCAT('c', FC_SEQ) WHERE FC_SEQ = ?`, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(id), nil
}

func placeholderFeedCategoryCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "new-" + hex.EncodeToString(buf), nil // 20 chars, fits FC_CODE
}

// Update renames a category and sets its app tab visibility. Code, order and the
// default flag never change here.
func (r *AdminFeedCategoryRepository) Update(seq int, name, openYN string) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_FEED_CATEGORY SET FC_NAME = ?, OPEN_YN = ?, UPD_DATE = NOW() WHERE FC_SEQ = ?
	`, name, openYN, seq)
	return err
}

// Reorder assigns SORT_ORDER i+1 to each seq in the given order in one transaction.
func (r *AdminFeedCategoryRepository) Reorder(seqs []int) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for i, seq := range seqs {
		if _, err := tx.Exec(`UPDATE ALUMNI_FEED_CATEGORY SET SORT_ORDER = ?, UPD_DATE = NOW() WHERE FC_SEQ = ?`, i+1, seq); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Delete removes a non-default category. With moveToSeq nil it deletes only
// when no post references the category and otherwise returns the referencing
// NOTICE post count without deleting. With moveToSeq set it first repoints every
// referencing post there; both happen in one transaction, so a post is never
// left on a deleted category.
func (r *AdminFeedCategoryRepository) Delete(seq int, moveToSeq *int) (int, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	// Lock the category row so a concurrent post save cannot race the count.
	var locked int
	if err := tx.Get(&locked, `SELECT FC_SEQ FROM ALUMNI_FEED_CATEGORY WHERE FC_SEQ = ? FOR UPDATE`, seq); err != nil {
		return 0, err
	}
	if moveToSeq == nil {
		var postCount int
		if err := tx.Get(&postCount, `
			SELECT COUNT(*) FROM WEO_BOARDBBS WHERE GATE = 'NOTICE' AND FEED_CATEGORY_SEQ = ?
		`, seq); err != nil {
			return 0, err
		}
		if postCount > 0 {
			return postCount, nil
		}
	} else if _, err := tx.Exec(`
		UPDATE WEO_BOARDBBS SET FEED_CATEGORY_SEQ = ? WHERE FEED_CATEGORY_SEQ = ?
	`, *moveToSeq, seq); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM ALUMNI_FEED_CATEGORY WHERE FC_SEQ = ? AND IS_DEFAULT = 'N'`, seq); err != nil {
		return 0, err
	}
	return 0, tx.Commit()
}
