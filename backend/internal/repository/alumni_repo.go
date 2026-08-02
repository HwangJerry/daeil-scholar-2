// alumni_repo.go — Approved-alumni search repository.
package repository

import (
	"database/sql"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type AlumniRepository struct {
	DB *sqlx.DB
}

func NewAlumniRepository(db *sqlx.DB) *AlumniRepository {
	return &AlumniRepository{DB: db}
}

func (r *AlumniRepository) Search(params model.AlumniSearchParams) ([]model.AlumniRecord, int, error) {
	where, args := buildAlumniFilters(params)

	countQuery := `
		SELECT COUNT(*) FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_SEQ > 0
	` + where

	var total int
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	limit := params.Size
	if limit <= 0 {
		limit = 20
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}

	query := `
		SELECT
			m.USR_SEQ, m.USR_NAME, m.USR_PHOTO,
			v.GRADUATION_YEAR, v.COHORT, v.DEPARTMENT,
			jc.AJC_NAME, m.USR_POSITION
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_SEQ > 0
	` + where + `
		ORDER BY m.USR_NAME ASC, m.USR_SEQ ASC
		LIMIT ? OFFSET ?
	`
	queryArgs := append(args, limit, (page-1)*limit)

	var records []model.AlumniRecord
	if err := r.DB.Select(&records, query, queryArgs...); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlumniRepository) GetDetail(viewerSeq, userSeq int) (*model.AlumniRecord, error) {
	var record model.AlumniRecord
	err := r.DB.Get(&record, `
		SELECT
			m.USR_SEQ, m.USR_NAME, m.USR_PHOTO,
			v.GRADUATION_YEAR, v.COHORT, v.DEPARTMENT,
			jc.AJC_NAME, m.USR_POSITION,
			m.USR_PHONE, m.USR_EMAIL,
			m.USR_PHONE_PUBLIC, m.USR_EMAIL_PUBLIC,
			EXISTS (
				SELECT 1 FROM ALUMNI_MEMBER_BLOCK b
				WHERE b.BLOCKER_USR_SEQ = ?
				  AND b.BLOCKED_USR_SEQ = m.USR_SEQ
			) AS BLOCKED_BY_ME
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_SEQ = ?
		LIMIT 1
	`, viewerSeq, userSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AlumniRepository) GetFilters() (*model.AlumniFilters, error) {
	graduationYears := make([]int, 0)
	if err := r.DB.Select(&graduationYears, `
		SELECT DISTINCT v.GRADUATION_YEAR
		FROM ALUMNI_VERIFICATION v
		WHERE v.STATUS = 'approved'
		  AND v.GRADUATION_YEAR IS NOT NULL
		ORDER BY v.GRADUATION_YEAR DESC
	`); err != nil {
		return nil, err
	}

	cohorts := make([]string, 0)
	if err := r.DB.Select(&cohorts, `
		SELECT DISTINCT v.COHORT
		FROM ALUMNI_VERIFICATION v
		WHERE v.STATUS = 'approved'
		  AND v.COHORT IS NOT NULL AND v.COHORT != ''
		ORDER BY
		  CASE WHEN v.COHORT REGEXP '^[0-9]+$' THEN 0 ELSE 1 END,
		  CASE WHEN v.COHORT REGEXP '^[0-9]+$' THEN CAST(v.COHORT AS UNSIGNED) END,
		  v.COHORT ASC
	`); err != nil {
		return nil, err
	}

	departments := make([]string, 0)
	if err := r.DB.Select(&departments, `
		SELECT DISTINCT v.DEPARTMENT
		FROM ALUMNI_VERIFICATION v
		WHERE v.STATUS = 'approved'
		  AND v.DEPARTMENT IS NOT NULL AND v.DEPARTMENT != ''
		ORDER BY v.DEPARTMENT ASC
	`); err != nil {
		return nil, err
	}

	jobCats, err := r.GetJobCategories()
	if err != nil {
		return nil, err
	}

	jobRoles := make([]string, 0)
	if err := r.DB.Select(&jobRoles, `
		SELECT DISTINCT m.USR_POSITION
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		WHERE v.STATUS = 'approved'
		  AND m.USR_POSITION IS NOT NULL AND m.USR_POSITION != ''
		ORDER BY m.USR_POSITION ASC
	`); err != nil {
		return nil, err
	}

	return &model.AlumniFilters{
		GraduationYears: graduationYears,
		Cohorts:         cohorts,
		Departments:     departments,
		JobCategories:   jobCats,
		JobRoles:        jobRoles,
	}, nil
}

// GetJobCategories returns all active job categories.
func (r *AlumniRepository) GetJobCategories() ([]model.JobCategory, error) {
	var cats []model.JobCategory
	if err := r.DB.Select(&cats, `
		SELECT AJC_SEQ, AJC_NAME
		FROM ALUMNI_JOB_CATEGORY
		WHERE OPEN_YN = 'Y'
		ORDER BY AJC_INDX ASC
	`); err != nil {
		return nil, err
	}
	return cats, nil
}

// GetWidgetPreview returns the first 5 alumni names (alphabetically) and the total alumni count.
func (r *AlumniRepository) GetWidgetPreview() ([]string, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*)
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		WHERE v.STATUS = 'approved'`); err != nil {
		return nil, 0, err
	}
	var names []string
	if err := r.DB.Select(&names, `
		SELECT m.USR_NAME
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		WHERE v.STATUS = 'approved'
		ORDER BY m.USR_NAME ASC, m.USR_SEQ ASC
		LIMIT 5`); err != nil {
		return nil, 0, err
	}
	return names, total, nil
}

// buildAlumniFilters constructs AND clauses for the WEO_MEMBER search query.
// Returns a where string (starting with AND) and the corresponding args.
func buildAlumniFilters(params model.AlumniSearchParams) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if params.Name != "" {
		clauses = append(clauses, "AND m.USR_NAME LIKE ?")
		args = append(args, "%"+params.Name+"%")
	}
	if params.GraduationYear > 0 {
		clauses = append(clauses, "AND v.GRADUATION_YEAR = ?")
		args = append(args, params.GraduationYear)
	}
	if params.Cohort != "" {
		clauses = append(clauses, "AND v.COHORT = ?")
		args = append(args, params.Cohort)
	}
	if params.Department != "" {
		clauses = append(clauses, "AND v.DEPARTMENT = ?")
		args = append(args, params.Department)
	}
	if params.JobCategory > 0 {
		clauses = append(clauses, "AND m.USR_JOB_CAT = ?")
		args = append(args, params.JobCategory)
	}
	if params.JobRole != "" {
		clauses = append(clauses, "AND m.USR_POSITION = ?")
		args = append(args, params.JobRole)
	}

	return strings.Join(clauses, " "), args
}
