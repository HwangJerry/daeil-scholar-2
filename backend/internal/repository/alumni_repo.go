// alumni_repo.go — Alumni search repository (FUNDAMENTAL_MEMBER-based queries)
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
	countQuery := `SELECT COUNT(*) FROM FUNDAMENTAL_MEMBER f
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
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
	argsWithPage := append([]interface{}{}, args...)
	argsWithPage = append(argsWithPage, limit, (page-1)*limit)
	query := `
		SELECT
			f.FM_SEQ,
			IFNULL(f.FM_NAME, m.USR_NAME)          AS FM_NAME,
			IFNULL(m.USR_FN,  f.FM_FN)             AS FM_FN,
			f.FM_DEPT,
			f.FM_COMPANY,
			f.FM_POSITION,
			IFNULL(m.USR_PHONE, f.FM_PHONE)        AS FM_PHONE,
			IFNULL(m.USR_EMAIL, f.FM_EMAIL)        AS FM_EMAIL,
			f.FM_SMS, f.FM_SPAM,
			m.USR_BIZ_NAME, m.USR_BIZ_DESC, m.USR_BIZ_ADDR,
			jc.AJC_NAME, jc.AJC_COLOR,
			m.USR_PHOTO, m.USR_SEQ,
			(SELECT GROUP_CONCAT(t.AUT_TAG ORDER BY t.AUT_INDX SEPARATOR ',')
			 FROM ALUMNI_USER_TAG t WHERE t.USR_SEQ = m.USR_SEQ) AS TAGS
		FROM FUNDAMENTAL_MEMBER f
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
	` + where + `
		ORDER BY IFNULL(f.FM_NAME, m.USR_NAME) ASC
		LIMIT ? OFFSET ?
	`
	var records []model.AlumniRecord
	if err := r.DB.Select(&records, query, argsWithPage...); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlumniRepository) GetFilters() (*model.AlumniFilters, error) {
	fnList := make([]string, 0)
	if err := r.DB.Select(&fnList, `
		SELECT DISTINCT IFNULL(m.USR_FN, f.FM_FN) AS FM_FN
		FROM FUNDAMENTAL_MEMBER f
		LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = f.FM_SEQ AND m.USR_STATUS != 'AAA'
		WHERE IFNULL(m.USR_FN, f.FM_FN) IS NOT NULL
		  AND IFNULL(m.USR_FN, f.FM_FN) != ''
		ORDER BY FM_FN + 0, FM_FN
	`); err != nil {
		return nil, err
	}
	deptList := make([]string, 0)
	if err := r.DB.Select(&deptList, `
		SELECT DISTINCT f.FM_DEPT
		FROM FUNDAMENTAL_MEMBER f
		WHERE f.FM_DEPT IS NOT NULL AND f.FM_DEPT != ''
		ORDER BY f.FM_DEPT
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

func buildAlumniFilters(params model.AlumniSearchParams) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}
	if params.FN != "" {
		clauses = append(clauses, "IFNULL(m.USR_FN, f.FM_FN) = ?")
		args = append(args, params.FN)
	}
	if params.Dept != "" {
		clauses = append(clauses, "f.FM_DEPT = ?")
		args = append(args, params.Dept)
	}
	if params.Name != "" {
		clauses = append(clauses, "(IFNULL(f.FM_NAME, m.USR_NAME) LIKE ?)")
		args = append(args, params.Name+"%")
	}
	if params.Company != "" {
		clauses = append(clauses, "f.FM_COMPANY LIKE ?")
		args = append(args, params.Company+"%")
	}
	if params.Position != "" {
		clauses = append(clauses, "f.FM_POSITION LIKE ?")
		args = append(args, params.Position+"%")
	}
	if params.JobCat > 0 {
		clauses = append(clauses, "m.USR_JOB_CAT = ?")
		args = append(args, params.JobCat)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
