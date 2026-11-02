// hall_of_fame_repo.go — SQL access for hall_of_fame_entry and hall_of_fame_import_batch.
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	hallOfFameAdminPageSize  = 50
	mysqlDuplicateEntryError = 1062
)

const hallOfFameColumns = `id, natural_key, display_name, kind, cohort, department, tier_man_won, contact_name,
	phone_hash, consent_public, visible, source, source_batch_id, created_at, updated_at, deleted_at`

// ErrHallOfFameEntryNotFound means the row is absent or soft-deleted.
var ErrHallOfFameEntryNotFound = errors.New("hall of fame entry not found")

// ErrHallOfFameDuplicateKey means a concurrent insert won the natural_key.
var ErrHallOfFameDuplicateKey = errors.New("hall of fame natural key already exists")

type HallOfFameRepository struct {
	DB *sqlx.DB
}

func NewHallOfFameRepository(db *sqlx.DB) *HallOfFameRepository {
	return &HallOfFameRepository{DB: db}
}

func (r *HallOfFameRepository) RunInTransaction(operation func(*sqlx.Tx) error) error {
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

// ListPublic returns exactly the rows the public wall may show.
func (r *HallOfFameRepository) ListPublic() ([]model.HallOfFameEntry, error) {
	entries := make([]model.HallOfFameEntry, 0)
	err := r.DB.Select(&entries, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry
		WHERE visible = 1 AND consent_public = 1 AND tier_man_won IS NOT NULL AND deleted_at IS NULL`)
	return entries, err
}

func hallOfFameAdminWhere(filter model.HallOfFameAdminFilter) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := make([]interface{}, 0, 4)
	if filter.NoTier {
		conditions = append(conditions, "tier_man_won IS NULL")
	} else if filter.Tier != nil {
		conditions = append(conditions, "tier_man_won = ?")
		args = append(args, *filter.Tier)
	}
	if filter.Kind != "" {
		conditions = append(conditions, "kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.Visible != nil {
		conditions = append(conditions, "visible = ?")
		args = append(args, *filter.Visible)
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
		conditions = append(conditions, "(display_name LIKE ? OR CAST(cohort AS CHAR) = ?)")
		args = append(args, "%"+escaped+"%", strings.TrimSuffix(query, "기"))
	}
	return strings.Join(conditions, " AND "), args
}

// ListAdmin returns one page, the filtered total and per-tier counts for operators.
func (r *HallOfFameRepository) ListAdmin(filter model.HallOfFameAdminFilter) ([]model.HallOfFameEntry, int, []model.HallOfFameTierCount, error) {
	where, args := hallOfFameAdminWhere(filter)
	var total int
	if err := r.DB.Get(&total, `SELECT COUNT(*) FROM hall_of_fame_entry WHERE `+where, args...); err != nil {
		return nil, 0, nil, err
	}
	counts := make([]model.HallOfFameTierCount, 0, len(model.HallOfFameTiersDescending)+1)
	if err := r.DB.Select(&counts, `SELECT tier_man_won, COUNT(*) AS entry_count FROM hall_of_fame_entry
		WHERE `+where+` GROUP BY tier_man_won ORDER BY tier_man_won DESC`, args...); err != nil {
		return nil, 0, nil, err
	}
	offset := (filter.Page - 1) * hallOfFameAdminPageSize
	entries := make([]model.HallOfFameEntry, 0, hallOfFameAdminPageSize)
	pageArgs := append(append([]interface{}{}, args...), hallOfFameAdminPageSize, offset)
	err := r.DB.Select(&entries, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry WHERE `+where+`
		ORDER BY tier_man_won IS NULL, tier_man_won DESC, cohort IS NULL, cohort, display_name, id LIMIT ? OFFSET ?`, pageArgs...)
	return entries, total, counts, err
}

func (r *HallOfFameRepository) AdminPageSize() int { return hallOfFameAdminPageSize }

func (r *HallOfFameRepository) FindActiveByID(id int64) (*model.HallOfFameEntry, error) {
	var entry model.HallOfFameEntry
	err := r.DB.Get(&entry, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry WHERE id = ? AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHallOfFameEntryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// FindByNaturalKey includes soft-deleted rows because the UNIQUE key does too.
func (r *HallOfFameRepository) FindByNaturalKey(naturalKey string) (*model.HallOfFameEntry, error) {
	var entry model.HallOfFameEntry
	err := r.DB.Get(&entry, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry WHERE natural_key = ?`, naturalKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *HallOfFameRepository) ListAllForImport() ([]model.HallOfFameEntry, error) {
	entries := make([]model.HallOfFameEntry, 0)
	err := r.DB.Select(&entries, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry`)
	return entries, err
}

// ListAllForImportTx locks every row so concurrent commits serialize.
func (r *HallOfFameRepository) ListAllForImportTx(tx *sqlx.Tx) ([]model.HallOfFameEntry, error) {
	entries := make([]model.HallOfFameEntry, 0)
	err := tx.Select(&entries, `SELECT `+hallOfFameColumns+` FROM hall_of_fame_entry FOR UPDATE`)
	return entries, err
}

func (r *HallOfFameRepository) FileAlreadyCommitted(fileSHA256 string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM hall_of_fame_import_batch WHERE file_sha256 = ?`, fileSHA256)
	return count > 0, err
}

const hallOfFameInsertSQL = `INSERT INTO hall_of_fame_entry
	(natural_key, display_name, kind, cohort, department, tier_man_won, contact_name, phone_hash,
	 consent_public, visible, source, source_batch_id, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 1, ?, ?, NOW(), NOW())`

func (r *HallOfFameRepository) Insert(entry model.HallOfFameEntry) (int64, error) {
	result, err := r.DB.Exec(hallOfFameInsertSQL, entry.NaturalKey, entry.DisplayName, entry.Kind, entry.Cohort, entry.Department,
		entry.TierManWon, entry.ContactName, entry.PhoneHash, entry.Source, entry.SourceBatchID)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryError {
			return 0, ErrHallOfFameDuplicateKey
		}
		return 0, err
	}
	return result.LastInsertId()
}

func (r *HallOfFameRepository) InsertTx(tx *sqlx.Tx, entry model.HallOfFameEntry) error {
	_, err := tx.Exec(hallOfFameInsertSQL, entry.NaturalKey, entry.DisplayName, entry.Kind, entry.Cohort, entry.Department,
		entry.TierManWon, entry.ContactName, entry.PhoneHash, entry.Source, entry.SourceBatchID)
	return err
}

const hallOfFameSaveSQL = `UPDATE hall_of_fame_entry SET display_name = ?, kind = ?, cohort = ?, department = ?,
	tier_man_won = ?, contact_name = ?, phone_hash = ?, consent_public = ?, visible = ?, source = ?, source_batch_id = ?,
	deleted_at = NULL, updated_at = NOW() WHERE id = ?`

// Save writes every mutable field and clears deleted_at, which also serves restores.
func (r *HallOfFameRepository) Save(entry model.HallOfFameEntry) error {
	_, err := r.DB.Exec(hallOfFameSaveSQL, entry.DisplayName, entry.Kind, entry.Cohort, entry.Department, entry.TierManWon,
		entry.ContactName, entry.PhoneHash, entry.ConsentPublic, entry.Visible, entry.Source, entry.SourceBatchID, entry.ID)
	return err
}

func (r *HallOfFameRepository) SaveTx(tx *sqlx.Tx, entry model.HallOfFameEntry) error {
	_, err := tx.Exec(hallOfFameSaveSQL, entry.DisplayName, entry.Kind, entry.Cohort, entry.Department, entry.TierManWon,
		entry.ContactName, entry.PhoneHash, entry.ConsentPublic, entry.Visible, entry.Source, entry.SourceBatchID, entry.ID)
	return err
}

// SoftDeleteManual is the operator delete. It marks the row manual, which makes it
// a tombstone: a later import must not bring back what an operator took down.
func (r *HallOfFameRepository) SoftDeleteManual(id int64) error {
	result, err := r.DB.Exec(`UPDATE hall_of_fame_entry SET deleted_at = NOW(), updated_at = NOW(), source = 'manual'
		WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrHallOfFameEntryNotFound
	}
	return nil
}

// SoftDeleteImportedTx removes an operator-selected removal candidate. Keeping
// source=excel lets the row return if a later spreadsheet lists the person again.
func (r *HallOfFameRepository) SoftDeleteImportedTx(tx *sqlx.Tx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`UPDATE hall_of_fame_entry SET deleted_at = NOW(), updated_at = NOW()
		WHERE id IN (?) AND source = 'excel' AND deleted_at IS NULL`, ids)
	if err != nil {
		return fmt.Errorf("build soft delete: %w", err)
	}
	_, err = tx.Exec(tx.Rebind(query), args...)
	return err
}

func (r *HallOfFameRepository) InsertBatchTx(tx *sqlx.Tx, batch model.HallOfFameImportBatch) error {
	_, err := tx.Exec(`INSERT INTO hall_of_fame_import_batch (id, file_sha256, uploaded_by, summary_json, committed_at)
		VALUES (?, ?, ?, ?, NOW())`, batch.ID, batch.FileSHA256, batch.UploadedBy, batch.SummaryJSON)
	return err
}
