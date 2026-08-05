// auth_principal_repo.go — Canonical mobile principal projection.
package repository

import (
	"database/sql"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type authPrincipalRow struct {
	USRSeq             int            `db:"USR_SEQ"`
	USRID              string         `db:"USR_ID"`
	USRName            string         `db:"USR_NAME"`
	Email              string         `db:"USR_EMAIL"`
	USRStatus          string         `db:"USR_STATUS"`
	AdminRole          sql.NullString `db:"ADMIN_ROLE"`
	VerificationStatus string         `db:"VERIFICATION_STATUS"`
	GraduationYear     sql.NullInt64  `db:"GRADUATION_YEAR"`
	Cohort             sql.NullString `db:"COHORT"`
	Department         sql.NullString `db:"DEPARTMENT"`
	RejectionReason    sql.NullString `db:"REJECTION_REASON"`
	SubmittedAt        sql.NullTime   `db:"SUBMITTED_AT"`
	ReviewedAt         sql.NullTime   `db:"REVIEWED_AT"`
}

func (r *AuthRepository) GetAuthPrincipalBySeq(usrSeq int) (*model.AuthUser, error) {
	var row authPrincipalRow
	err := r.DB.Get(&row, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, COALESCE(m.USR_EMAIL, '') AS USR_EMAIL, m.USR_STATUS,
		       ar.ADMIN_ROLE,
		       v.STATUS AS VERIFICATION_STATUS,
		       v.GRADUATION_YEAR,
		       v.COHORT,
		       v.DEPARTMENT,
		       v.REJECTION_REASON,
		       v.SUBMITTED_AT,
		       v.REVIEWED_AT
		FROM WEO_MEMBER m
		JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
		LEFT JOIN ALUMNI_ADMIN_ROLE ar ON ar.USR_SEQ = m.USR_SEQ
		WHERE m.USR_SEQ = ?
		LIMIT 1
	`, usrSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	principal := &model.AuthUser{
		USRSeq:    row.USRSeq,
		USRID:     row.USRID,
		USRName:   row.USRName,
		Email:     row.Email,
		USRStatus: row.USRStatus,
		Verification: model.AlumniVerification{
			Status:          model.VerificationStatus(row.VerificationStatus),
			GraduationYear:  nullableInt(row.GraduationYear),
			Cohort:          nullableString(row.Cohort),
			Department:      nullableString(row.Department),
			RejectionReason: nullableString(row.RejectionReason),
			SubmittedAt:     nullableTime(row.SubmittedAt),
			ReviewedAt:      nullableTime(row.ReviewedAt),
		},
	}
	if row.AdminRole.Valid {
		role := model.AdminRole(row.AdminRole.String)
		principal.AdminRole = &role
	}
	return principal, nil
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}
