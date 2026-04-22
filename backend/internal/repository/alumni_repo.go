// alumni_repo.go — Alumni search repository (WEO_MEMBER only)
package repository

import (
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
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE m.USR_STATUS IN ('CCC', 'ZZZ')
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
			m.USR_SEQ, m.USR_NAME, m.USR_FN, m.USR_DEPT,
			m.USR_BIZ_NAME, m.USR_BIZ_DESC, m.USR_BIZ_ADDR,
			m.USR_POSITION,
			m.USR_PHONE, m.USR_EMAIL, m.USR_PHOTO,
			m.USR_NICK, m.USR_BIZ_CARD,
			m.USR_PHONE_PUBLIC, m.USR_EMAIL_PUBLIC,
			jc.AJC_NAME,
			(SELECT GROUP_CONCAT(t.AUT_TAG ORDER BY t.AUT_INDX SEPARATOR ',')
			 FROM ALUMNI_USER_TAG t WHERE t.USR_SEQ = m.USR_SEQ) AS TAGS
		FROM WEO_MEMBER m
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
		WHERE m.USR_STATUS IN ('CCC', 'ZZZ')
		  AND m.USR_SEQ > 0
	` + where + `
		ORDER BY m.USR_NAME ASC
		LIMIT ? OFFSET ?
	`
	queryArgs := append(args, limit, (page-1)*limit)

	var records []model.AlumniRecord
	if err := r.DB.Select(&records, query, queryArgs...); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlumniRepository) GetFilters() (*model.AlumniFilters, error) {
	fnList := make([]string, 0)
	if err := r.DB.Select(&fnList, `
		SELECT DISTINCT USR_FN AS fn
		FROM WEO_MEMBER
		WHERE USR_STATUS IN ('CCC', 'ZZZ')
		  AND USR_SEQ > 0
		  AND USR_FN IS NOT NULL AND USR_FN != ''
		ORDER BY fn + 0, fn
	`); err != nil {
		return nil, err
	}

	deptList := make([]string, 0)
	if err := r.DB.Select(&deptList, `
		SELECT DISTINCT USR_DEPT AS dept
		FROM WEO_MEMBER
		WHERE USR_STATUS IN ('CCC', 'ZZZ')
		  AND USR_SEQ > 0
		  AND USR_DEPT IS NOT NULL AND USR_DEPT != ''
		ORDER BY dept
	`); err != nil {
		return nil, err
	}

	jobCats, err := r.GetJobCategories()
	if err != nil {
		return nil, err
	}
	return &model.AlumniFilters{FNList: fnList, DeptList: deptList, JobCategories: jobCats}, nil
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

// GetWeeklyCount returns the number of alumni members who registered in the last 7 days.
func (r *AlumniRepository) GetWeeklyCount() (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_MEMBER m
		WHERE m.USR_STATUS IN ('CCC', 'ZZZ')
		  AND m.REG_DATE > DATE_SUB(NOW(), INTERVAL 7 DAY)
	`)
	return count, err
}

// GetWidgetPreview returns the first 5 alumni names (alphabetically) and the total alumni count.
func (r *AlumniRepository) GetWidgetPreview() ([]string, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_STATUS IN ('CCC', 'ZZZ')`); err != nil {
		return nil, 0, err
	}
	var names []string
	if err := r.DB.Select(&names, `
		SELECT USR_NAME FROM WEO_MEMBER
		WHERE USR_STATUS IN ('CCC', 'ZZZ')
		ORDER BY USR_NAME ASC LIMIT 5`); err != nil {
		return nil, 0, err
	}
	return names, total, nil
}

// buildAlumniFilters constructs AND clauses for the WEO_MEMBER search query.
// Returns a where string (starting with AND) and the corresponding args.
func buildAlumniFilters(params model.AlumniSearchParams) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if params.FN != "" {
		clauses = append(clauses, "AND m.USR_FN = ?")
		args = append(args, params.FN)
	}
	if params.Dept != "" {
		clauses = append(clauses, "AND m.USR_DEPT = ?")
		args = append(args, params.Dept)
	}
	for _, kw := range strings.Fields(params.Name) {
		clause, kwArgs := buildKeywordClause(kw)
		clauses = append(clauses, clause)
		args = append(args, kwArgs...)
	}
	if params.Company != "" {
		clauses = append(clauses, "AND m.USR_BIZ_NAME LIKE ?")
		args = append(args, params.Company+"%")
	}
	if params.JobCat > 0 {
		clauses = append(clauses, "AND m.USR_JOB_CAT = ?")
		args = append(args, params.JobCat)
	}

	return strings.Join(clauses, " "), args
}

// buildKeywordClause returns a single AND clause matching the keyword against
// either USR_NAME or any of the user's tags (ALUMNI_USER_TAG). Uses EXISTS
// subquery for MariaDB 10.1 compatibility (no CTE/window functions).
func buildKeywordClause(keyword string) (string, []interface{}) {
	like := "%" + keyword + "%"
	clause := "AND (m.USR_NAME LIKE ? OR EXISTS (" +
		"SELECT 1 FROM ALUMNI_USER_TAG t " +
		"WHERE t.USR_SEQ = m.USR_SEQ AND t.AUT_TAG LIKE ?" +
		"))"
	return clause, []interface{}{like, like}
}
