// auth_principal.go — Loads canonical authenticated principals and authorization state.
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type authPrincipalRow struct {
	USRSeq             int            `db:"USR_SEQ"`
	USRID              string         `db:"USR_ID"`
	USRName            string         `db:"USR_NAME"`
	USRStatus          string         `db:"USR_STATUS"`
	Email              string         `db:"EMAIL"`
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
		SELECT
			m.USR_SEQ,
			m.USR_ID,
			m.USR_NAME,
			m.USR_STATUS,
			COALESCE(m.USR_EMAIL, '') AS EMAIL,
			ar.ADMIN_ROLE,
			COALESCE(v.STATUS, 'unsubmitted') AS VERIFICATION_STATUS,
			v.GRADUATION_YEAR,
			v.COHORT,
			v.DEPARTMENT,
			v.REJECTION_REASON,
			v.SUBMITTED_AT,
			v.REVIEWED_AT
		FROM WEO_MEMBER m
		LEFT JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = m.USR_SEQ
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
	return mapAuthPrincipal(row)
}

func mapAuthPrincipal(row authPrincipalRow) (*model.AuthUser, error) {
	status := model.VerificationStatus(row.VerificationStatus)
	if !status.Valid() {
		return nil, fmt.Errorf("invalid alumni verification status %q", row.VerificationStatus)
	}

	var role *model.AdminRole
	if row.AdminRole.Valid {
		value := model.AdminRole(row.AdminRole.String)
		if !value.Valid() {
			return nil, fmt.Errorf("invalid admin role %q", row.AdminRole.String)
		}
		role = &value
	}

	return &model.AuthUser{
		USRSeq:    row.USRSeq,
		USRID:     row.USRID,
		USRName:   row.USRName,
		Email:     row.Email,
		AdminRole: role,
		Verification: model.AlumniVerification{
			Status:          status,
			GraduationYear:  nullableInt(row.GraduationYear),
			Cohort:          nullableString(row.Cohort),
			Department:      nullableString(row.Department),
			RejectionReason: nullableString(row.RejectionReason),
			SubmittedAt:     nullableTime(row.SubmittedAt),
			ReviewedAt:      nullableTime(row.ReviewedAt),
		},
		USRStatus: row.USRStatus,
	}, nil
}

func nullableInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	converted := value.String
	return &converted
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	converted := value.Time
	return &converted
}
