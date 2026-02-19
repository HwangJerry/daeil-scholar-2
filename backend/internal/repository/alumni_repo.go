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
		INNER JOIN WEO_MEMBER m ON f.FM_SEQ = m.USR_SEQ ` + where
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
		SELECT f.FM_SEQ, f.FM_NAME, f.FM_FN, f.FM_DEPT, f.FM_COMPANY, f.FM_POSITION,
			IFNULL(f.FM_PHONE, m.USR_PHONE) AS FM_PHONE,
			IFNULL(f.FM_EMAIL, m.USR_EMAIL) AS FM_EMAIL,
			f.FM_SMS, f.FM_SPAM,
			m.USR_BIZ_NAME, m.USR_BIZ_DESC, m.USR_BIZ_ADDR,
			jc.AJC_NAME, jc.AJC_COLOR,
			m.USR_PHOTO, m.USR_SEQ,
			(SELECT GROUP_CONCAT(t.AUT_TAG ORDER BY t.AUT_INDX SEPARATOR ',')
			 FROM ALUMNI_USER_TAG t WHERE t.USR_SEQ = m.USR_SEQ) AS TAGS
		FROM FUNDAMENTAL_MEMBER f
		INNER JOIN WEO_MEMBER m ON f.FM_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_JOB_CATEGORY jc ON m.USR_JOB_CAT = jc.AJC_SEQ
	` + where + `
		ORDER BY f.FM_NAME ASC
		LIMIT ? OFFSET ?
	`
	var records []model.AlumniRecord
	if err := r.DB.Select(&records, query, argsWithPage...); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlumniRepository) GetFilters() (*model.AlumniFilters, error) {
	var fnList []string
	if err := r.DB.Select(&fnList, `
		SELECT DISTINCT FM_FN FROM FUNDAMENTAL_MEMBER ORDER BY FM_FN
	`); err != nil {
		return nil, err
	}
	var deptList []string
	if err := r.DB.Select(&deptList, `
		SELECT DISTINCT FM_DEPT FROM FUNDAMENTAL_MEMBER WHERE FM_DEPT != '' ORDER BY FM_DEPT
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
// Joins FUNDAMENTAL_MEMBER for consistency with the alumni search. MariaDB 10.1.38 compatible.
func (r *AlumniRepository) GetWeeklyCount() (int, error) {
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*) FROM WEO_MEMBER m
		INNER JOIN FUNDAMENTAL_MEMBER f ON f.FM_SEQ = m.USR_SEQ
		WHERE m.REG_DATE > DATE_SUB(NOW(), INTERVAL 7 DAY)
	`)
	return count, err
}

func buildAlumniFilters(params model.AlumniSearchParams) (string, []interface{}) {
	clauses := []string{"1=1"}
	args := []interface{}{}
	if params.FN != "" {
		clauses = append(clauses, "f.FM_FN = ?")
		args = append(args, params.FN)
	}
	if params.Dept != "" {
		clauses = append(clauses, "f.FM_DEPT = ?")
		args = append(args, params.Dept)
	}
	if params.Name != "" {
		clauses = append(clauses, "f.FM_NAME LIKE ?")
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
	return "WHERE " + strings.Join(clauses, " AND "), args
}
