// admin_job_category_repo.go — CRUD queries for ALUMNI_JOB_CATEGORY admin operations
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AdminJobCategoryRepository struct {
	DB *sqlx.DB
}

func NewAdminJobCategoryRepository(db *sqlx.DB) *AdminJobCategoryRepository {
	return &AdminJobCategoryRepository{DB: db}
}

// GetAll returns all job categories regardless of OPEN_YN status.
func (r *AdminJobCategoryRepository) GetAll() ([]model.AdminJobCategory, error) {
	var cats []model.AdminJobCategory
	err := r.DB.Select(&cats, `
		SELECT AJC_SEQ, AJC_NAME, AJC_INDX, OPEN_YN
		FROM ALUMNI_JOB_CATEGORY
		ORDER BY AJC_INDX ASC
	`)
	return cats, err
}

// Insert inserts a new job category and returns the new primary key.
func (r *AdminJobCategoryRepository) Insert(req model.AdminJobCategoryUpsert) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO ALUMNI_JOB_CATEGORY (AJC_NAME, AJC_INDX, OPEN_YN, REG_DATE)
		VALUES (?, ?, ?, NOW())
	`, req.Name, req.Index, req.OpenYN)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update updates an existing job category by seq.
func (r *AdminJobCategoryRepository) Update(seq int, req model.AdminJobCategoryUpsert) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_JOB_CATEGORY
		SET AJC_NAME = ?, AJC_INDX = ?, OPEN_YN = ?
		WHERE AJC_SEQ = ?
	`, req.Name, req.Index, req.OpenYN, seq)
	return err
}

// SoftDelete sets OPEN_YN = 'N' for the given seq.
func (r *AdminJobCategoryRepository) SoftDelete(seq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_JOB_CATEGORY SET OPEN_YN = 'N' WHERE AJC_SEQ = ?
	`, seq)
	return err
}
