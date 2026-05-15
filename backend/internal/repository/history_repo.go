// history_repo.go — CRUD queries for HISTORY_ENTRY table
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type HistoryRepository struct {
	DB *sqlx.DB
}

func NewHistoryRepository(db *sqlx.DB) *HistoryRepository {
	return &HistoryRepository{DB: db}
}

// GetAll returns all history entries ordered newest first, then by sort order.
func (r *HistoryRepository) GetAll() ([]model.HistoryEntry, error) {
	var entries []model.HistoryEntry
	err := r.DB.Select(&entries, `
		SELECT HE_SEQ, DATE_FORMAT(HE_EVENT_DATE, '%Y-%m-%d') AS HE_EVENT_DATE,
		       HE_TEXT, HE_SORT_ORDER
		FROM HISTORY_ENTRY
		ORDER BY HE_EVENT_DATE DESC, HE_SORT_ORDER ASC
	`)
	return entries, err
}

// GetBySeq returns a single entry by primary key.
func (r *HistoryRepository) GetBySeq(seq int) (*model.HistoryEntry, error) {
	var entry model.HistoryEntry
	err := r.DB.Get(&entry, `
		SELECT HE_SEQ, DATE_FORMAT(HE_EVENT_DATE, '%Y-%m-%d') AS HE_EVENT_DATE,
		       HE_TEXT, HE_SORT_ORDER
		FROM HISTORY_ENTRY
		WHERE HE_SEQ = ?
	`, seq)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Insert inserts a new history entry and returns the new primary key.
func (r *HistoryRepository) Insert(req model.HistoryUpsertRequest) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO HISTORY_ENTRY (HE_EVENT_DATE, HE_TEXT, HE_SORT_ORDER, REG_DATE)
		VALUES (?, ?, ?, NOW())
	`, req.EventDate, req.Text, req.SortOrder)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update updates an existing history entry.
func (r *HistoryRepository) Update(seq int, req model.HistoryUpsertRequest) error {
	_, err := r.DB.Exec(`
		UPDATE HISTORY_ENTRY
		SET HE_EVENT_DATE = ?, HE_TEXT = ?, HE_SORT_ORDER = ?, MOD_DATE = NOW()
		WHERE HE_SEQ = ?
	`, req.EventDate, req.Text, req.SortOrder, seq)
	return err
}

// Delete permanently deletes a history entry.
func (r *HistoryRepository) Delete(seq int) error {
	_, err := r.DB.Exec(`DELETE FROM HISTORY_ENTRY WHERE HE_SEQ = ?`, seq)
	return err
}
