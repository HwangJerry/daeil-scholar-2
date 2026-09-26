// admin_operator_repo.go — Grants, changes and revokes ALUMNI_ADMIN_ROLE rows while keeping at least one root
package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrOperatorTargetIneligible = errors.New("operator target is not an active member")
	ErrOperatorNotFound         = errors.New("member has no admin role")
	ErrLastRootAdmin            = errors.New("at least one root admin must remain")
)

// Member statuses that cannot hold an admin role (탈퇴, 휴면, 정지).
var inactiveMemberStatuses = map[string]bool{"AAA": true, "ABA": true, "ACA": true}

type AdminOperatorRepository struct {
	DB *sqlx.DB
}

func NewAdminOperatorRepository(db *sqlx.DB) *AdminOperatorRepository {
	return &AdminOperatorRepository{DB: db}
}

func (r *AdminOperatorRepository) ListOperators() ([]model.AdminOperator, error) {
	rows := []model.AdminOperator{}
	err := r.DB.Select(&rows, `
		SELECT ar.USR_SEQ, m.USR_ID, m.USR_NAME, ar.ADMIN_ROLE, ar.CREATED_AT, ar.UPDATED_AT,
		       IFNULL(u.USR_NAME, '') AS UPDATED_BY_NAME
		FROM ALUMNI_ADMIN_ROLE ar
		JOIN WEO_MEMBER m ON m.USR_SEQ = ar.USR_SEQ
		LEFT JOIN WEO_MEMBER u ON u.USR_SEQ = ar.UPDATED_BY
		ORDER BY FIELD(ar.ADMIN_ROLE, 'root', 'operator'), m.USR_NAME, ar.USR_SEQ
	`)
	return rows, err
}

// SetOperatorRole grants role to an active member, or changes an existing
// holder's role. Demoting the only root is refused.
func (r *AdminOperatorRepository) SetOperatorRole(actorSeq, targetSeq int, role model.AdminRole) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status sql.NullString
	err = tx.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ = ? FOR UPDATE`, targetSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOperatorTargetIneligible
	}
	if err != nil {
		return err
	}
	if inactiveMemberStatuses[status.String] {
		return ErrOperatorTargetIneligible
	}

	current, rootCount, err := lockAdminRolesTx(tx, targetSeq)
	if err != nil {
		return err
	}
	if current == model.AdminRoleRoot && role != model.AdminRoleRoot && rootCount <= 1 {
		return ErrLastRootAdmin
	}

	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_ADMIN_ROLE (USR_SEQ, ADMIN_ROLE, CREATED_AT, UPDATED_AT, CREATED_BY, UPDATED_BY)
		VALUES (?, ?, NOW(), NOW(), ?, ?)
		ON DUPLICATE KEY UPDATE
			ADMIN_ROLE = VALUES(ADMIN_ROLE), UPDATED_AT = NOW(), UPDATED_BY = VALUES(UPDATED_BY)
	`, targetSeq, string(role), actorSeq, actorSeq); err != nil {
		return err
	}
	return tx.Commit()
}

// RevokeOperator removes a member's admin role. Removing the only root is refused.
func (r *AdminOperatorRepository) RevokeOperator(targetSeq int) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	current, rootCount, err := lockAdminRolesTx(tx, targetSeq)
	if err != nil {
		return err
	}
	if current == "" {
		return ErrOperatorNotFound
	}
	if current == model.AdminRoleRoot && rootCount <= 1 {
		return ErrLastRootAdmin
	}
	if _, err := tx.Exec(`DELETE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = ?`, targetSeq); err != nil {
		return err
	}
	return tx.Commit()
}

// lockAdminRolesTx locks every root row and the target's row so two roots
// demoting each other concurrently cannot leave the system without a root.
// It returns the target's current role ("" when none) and the root count.
func lockAdminRolesTx(tx *sqlx.Tx, targetSeq int) (model.AdminRole, int, error) {
	var roots []int
	if err := tx.Select(&roots, `
		SELECT USR_SEQ FROM ALUMNI_ADMIN_ROLE WHERE ADMIN_ROLE = 'root' ORDER BY USR_SEQ FOR UPDATE
	`); err != nil {
		return "", 0, err
	}
	var current string
	err := tx.Get(&current, `SELECT ADMIN_ROLE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = ? FOR UPDATE`, targetSeq)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", 0, err
	}
	return model.AdminRole(current), len(roots), nil
}
