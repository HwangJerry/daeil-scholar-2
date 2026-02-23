// alumni_repo.go — Alumni search repository (UNION ALL: FUNDAMENTAL_MEMBER + WEO_MEMBER-only)
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

// alumniFilterClauses holds separate WHERE/AND clauses for Part A and Part B of the UNION ALL query.
type alumniFilterClauses struct {
	fundWhere string
	fundArgs  []interface{}
	weoWhere  string
	weoArgs   []interface{}
}

func (r *AlumniRepository) Search(params model.AlumniSearchParams) ([]model.AlumniRecord, int, error) {
	filters := buildAlumniFilters(params)

	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT f.FM_SEQ
			FROM FUNDAMENTAL_MEMBER f
			LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
			` + filters.fundWhere + `
			UNION ALL
			SELECT m.USR_SEQ
			FROM WEO_MEMBER m
			WHERE m.USR_STATUS != 'AAA'
			  AND NOT EXISTS (SELECT 1 FROM FUNDAMENTAL_MEMBER f WHERE f.FM_SEQ = m.USR_SEQ)
			` + filters.weoWhere + `
		) AS combined
	`
	countArgs := append(filters.fundArgs, filters.weoArgs...)
	var total int
	if err := r.DB.Get(&total, countQuery, countArgs...); err != nil {
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
		SELECT * FROM (
			SELECT
				f.FM_SEQ                                    AS FM_SEQ,
				IFNULL(f.FM_NAME, m.USR_NAME)               AS FM_NAME,
				IFNULL(m.USR_FN,  f.FM_FN)                  AS FM_FN,
				IFNULL(m.USR_DEPT, f.FM_DEPT)               AS FM_DEPT,
				f.FM_COMPANY,
				f.FM_POSITION,
				IFNULL(m.USR_PHONE, f.FM_PHONE)             AS FM_PHONE,
				IFNULL(m.USR_EMAIL, f.FM_EMAIL)             AS FM_EMAIL,
				f.FM_SMS, f.FM_SPAM,
				m.USR_BIZ_NAME, m.USR_BIZ_DESC, m.USR_BIZ_ADDR,
				jc.AJC_NAME, jc.AJC_COLOR,
				m.USR_PHOTO, m.USR_SEQ,
				(SELECT GROUP_CONCAT(t.AUT_TAG ORDER BY t.AUT_INDX SEPARATOR ',')
				 FROM ALUMNI_USER_TAG t WHERE t.USR_SEQ = m.USR_SEQ) AS TAGS
			FROM FUNDAMENTAL_MEMBER f
			LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
			LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
			` + filters.fundWhere + `
			UNION ALL
			SELECT
				NULL                AS FM_SEQ,
				m.USR_NAME          AS FM_NAME,
				m.USR_FN            AS FM_FN,
				m.USR_DEPT          AS FM_DEPT,
				m.USR_BIZ_NAME      AS FM_COMPANY,
				NULL                AS FM_POSITION,
				m.USR_PHONE         AS FM_PHONE,
				m.USR_EMAIL         AS FM_EMAIL,
				'N'                 AS FM_SMS,
				'N'                 AS FM_SPAM,
				m.USR_BIZ_NAME, m.USR_BIZ_DESC, m.USR_BIZ_ADDR,
				jc.AJC_NAME, jc.AJC_COLOR,
				m.USR_PHOTO, m.USR_SEQ,
				(SELECT GROUP_CONCAT(t.AUT_TAG ORDER BY t.AUT_INDX SEPARATOR ',')
				 FROM ALUMNI_USER_TAG t WHERE t.USR_SEQ = m.USR_SEQ) AS TAGS
			FROM WEO_MEMBER m
			LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
			WHERE m.USR_STATUS != 'AAA'
			  AND NOT EXISTS (SELECT 1 FROM FUNDAMENTAL_MEMBER f WHERE f.FM_SEQ = m.USR_SEQ)
			` + filters.weoWhere + `
		) AS combined
		ORDER BY FM_NAME ASC
		LIMIT ? OFFSET ?
	`
	queryArgs := append(filters.fundArgs, filters.weoArgs...)
	queryArgs = append(queryArgs, limit, (page-1)*limit)

	var records []model.AlumniRecord
	if err := r.DB.Select(&records, query, queryArgs...); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlumniRepository) GetFilters() (*model.AlumniFilters, error) {
	fnList := make([]string, 0)
	if err := r.DB.Select(&fnList, `
		SELECT DISTINCT fn FROM (
			SELECT IFNULL(m.USR_FN, f.FM_FN) AS fn
			FROM FUNDAMENTAL_MEMBER f
			LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
			WHERE IFNULL(m.USR_FN, f.FM_FN) IS NOT NULL
			  AND IFNULL(m.USR_FN, f.FM_FN) != ''
			UNION
			SELECT m.USR_FN AS fn
			FROM WEO_MEMBER m
			WHERE m.USR_STATUS != 'AAA'
			  AND m.USR_FN IS NOT NULL AND m.USR_FN != ''
			  AND NOT EXISTS (SELECT 1 FROM FUNDAMENTAL_MEMBER f WHERE f.FM_SEQ = m.USR_SEQ)
		) AS all_fn ORDER BY fn + 0, fn
	`); err != nil {
		return nil, err
	}
	deptList := make([]string, 0)
	if err := r.DB.Select(&deptList, `
		SELECT DISTINCT dept FROM (
			SELECT f.FM_DEPT AS dept
			FROM FUNDAMENTAL_MEMBER f
			WHERE f.FM_DEPT IS NOT NULL AND f.FM_DEPT != ''
			UNION
			SELECT m.USR_DEPT AS dept
			FROM WEO_MEMBER m
			WHERE m.USR_STATUS != 'AAA' AND m.USR_DEPT IS NOT NULL AND m.USR_DEPT != ''
			  AND NOT EXISTS (SELECT 1 FROM FUNDAMENTAL_MEMBER f WHERE f.FM_SEQ = m.USR_SEQ)
		) AS all_depts ORDER BY dept
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
		SELECT AJC_SEQ, AJC_NAME, AJC_COLOR
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
		WHERE m.USR_STATUS != 'AAA'
		  AND m.REG_DATE > DATE_SUB(NOW(), INTERVAL 7 DAY)
	`)
	return count, err
}

// GetWidgetPreview returns the first 5 alumni names (alphabetically) and the total alumni count.
func (r *AlumniRepository) GetWidgetPreview() ([]string, int, error) {
	var total int
	if err := r.DB.Get(&total, `
		SELECT COUNT(*) FROM WEO_MEMBER m
		LEFT JOIN FUNDAMENTAL_MEMBER f ON f.FM_SEQ = m.USR_SEQ
		WHERE m.USR_STATUS != 'AAA'`); err != nil {
		return nil, 0, err
	}
	var names []string
	if err := r.DB.Select(&names, `
		SELECT IFNULL(f.FM_NAME, m.USR_NAME) FROM WEO_MEMBER m
		LEFT JOIN FUNDAMENTAL_MEMBER f ON f.FM_SEQ = m.USR_SEQ
		WHERE m.USR_STATUS != 'AAA'
		ORDER BY IFNULL(f.FM_NAME, m.USR_NAME) ASC LIMIT 5`); err != nil {
		return nil, 0, err
	}
	return names, total, nil
}

// buildAlumniFilters constructs separate filter clauses for Part A (FUNDAMENTAL_MEMBER-based)
// and Part B (WEO_MEMBER-only) of the UNION ALL search query.
func buildAlumniFilters(params model.AlumniSearchParams) alumniFilterClauses {
	var fundClauses, weoClauses []string
	var fundArgs, weoArgs []interface{}

	if params.FN != "" {
		fundClauses = append(fundClauses, "IFNULL(m.USR_FN, f.FM_FN) = ?")
		fundArgs = append(fundArgs, params.FN)
		weoClauses = append(weoClauses, "AND m.USR_FN = ?")
		weoArgs = append(weoArgs, params.FN)
	}
	if params.Dept != "" {
		fundClauses = append(fundClauses, "IFNULL(m.USR_DEPT, f.FM_DEPT) = ?")
		fundArgs = append(fundArgs, params.Dept)
		weoClauses = append(weoClauses, "AND m.USR_DEPT = ?")
		weoArgs = append(weoArgs, params.Dept)
	}
	if params.Name != "" {
		fundClauses = append(fundClauses, "IFNULL(f.FM_NAME, m.USR_NAME) LIKE ?")
		fundArgs = append(fundArgs, params.Name+"%")
		weoClauses = append(weoClauses, "AND m.USR_NAME LIKE ?")
		weoArgs = append(weoArgs, params.Name+"%")
	}
	if params.Company != "" {
		fundClauses = append(fundClauses, "f.FM_COMPANY LIKE ?")
		fundArgs = append(fundArgs, params.Company+"%")
		weoClauses = append(weoClauses, "AND m.USR_BIZ_NAME LIKE ?")
		weoArgs = append(weoArgs, params.Company+"%")
	}
	if params.Position != "" {
		fundClauses = append(fundClauses, "f.FM_POSITION LIKE ?")
		fundArgs = append(fundArgs, params.Position+"%")
		// Part B has no position field — omit
	}
	if params.JobCat > 0 {
		fundClauses = append(fundClauses, "m.USR_JOB_CAT = ?")
		fundArgs = append(fundArgs, params.JobCat)
		weoClauses = append(weoClauses, "AND m.USR_JOB_CAT = ?")
		weoArgs = append(weoArgs, params.JobCat)
	}

	fundWhere := ""
	if len(fundClauses) > 0 {
		fundWhere = "WHERE " + strings.Join(fundClauses, " AND ")
	}
	weoWhere := strings.Join(weoClauses, " ")

	return alumniFilterClauses{
		fundWhere: fundWhere,
		fundArgs:  fundArgs,
		weoWhere:  weoWhere,
		weoArgs:   weoArgs,
	}
}
