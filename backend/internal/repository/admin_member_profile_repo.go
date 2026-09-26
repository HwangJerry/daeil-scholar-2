// admin_member_profile_repo.go — Admin correction of a member's name, contact and academic fields
package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
)

var ErrMemberNotFound = errors.New("member not found")

// UpdateMemberProfile writes an administrator's correction in one transaction.
// The phone claim moves with the phone so AUTH_PHONE_CLAIM stays authoritative.
// Cohort and department follow the member profile rule: WEO_MEMBER is updated,
// and an approved verification's current COHORT/DEPARTMENT are kept in sync,
// while review history and APPROVED_* snapshots are left untouched.
// The caller passes an already-normalized phone ("" keeps the stored phone).
func (r *AdminMemberRepository) UpdateMemberProfile(seq int, req model.AdminMemberProfileUpdate) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentPhone sql.NullString
	err = tx.Get(&currentPhone, `SELECT USR_PHONE FROM WEO_MEMBER WHERE USR_SEQ = ? FOR UPDATE`, seq)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMemberNotFound
	}
	if err != nil {
		return err
	}

	phone := currentPhone.String
	if req.USRPhone != "" {
		claimsReady, err := detectPhoneClaimsInWriteTransaction(tx)
		if err != nil {
			return err
		}
		if claimsReady {
			if err := transferPhoneClaim(tx, seq, currentPhone.String, req.USRPhone); err != nil {
				return err
			}
		} else if len(req.USRPhone) > model.LegacyPhoneDigitsLimit {
			return ErrInvalidPhone
		}
		phone = req.USRPhone
	}

	if _, err := tx.Exec(`
		UPDATE WEO_MEMBER
		LEFT JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = WEO_MEMBER.USR_SEQ AND v.STATUS = 'approved'
		SET WEO_MEMBER.USR_NAME = ?,
			WEO_MEMBER.USR_PHONE = ?,
			WEO_MEMBER.USR_EMAIL = NULLIF(?, ''),
			WEO_MEMBER.USR_FN = COALESCE(NULLIF(?, ''), WEO_MEMBER.USR_FN),
			WEO_MEMBER.USR_DEPT = COALESCE(NULLIF(?, ''), WEO_MEMBER.USR_DEPT),
			WEO_MEMBER.EDT_DATE = NOW(),
			v.COHORT = COALESCE(NULLIF(?, ''), v.COHORT),
			v.DEPARTMENT = COALESCE(NULLIF(?, ''), v.DEPARTMENT)
		WHERE WEO_MEMBER.USR_SEQ = ?
	`, req.USRName, phone, req.USREmail, req.USRFN, req.USRDept, req.USRFN, req.USRDept, seq); err != nil {
		return err
	}
	return tx.Commit()
}
