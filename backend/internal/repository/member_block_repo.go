package repository

// Approval is canonical in ALUMNI_VERIFICATION; new approved members retain BBB.
// The active login statuses BBB/CCC/ZZZ exclude withdrawn and inactive accounts.

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrMemberBlockTargetNotApproved = errors.New("member block target is not an approved alumnus")

// MemberBlockRepository persists directional member blocks.
type MemberBlockRepository struct {
	DB *sqlx.DB
}

func NewMemberBlockRepository(db *sqlx.DB) *MemberBlockRepository {
	return &MemberBlockRepository{DB: db}
}

func (r *MemberBlockRepository) IsApprovedAlumni(userSeq int) (bool, error) {
	var approved bool
	if err := r.DB.Get(&approved, `
		SELECT EXISTS(
			SELECT 1
			FROM ALUMNI_VERIFICATION v
			JOIN WEO_MEMBER m ON m.USR_SEQ = v.USR_SEQ
			WHERE v.USR_SEQ = ? AND v.STATUS = 'approved'
			  AND m.USR_STATUS IN ('BBB','CCC','ZZZ')
		)
	`, userSeq); err != nil {
		return false, err
	}
	return approved, nil
}

func (r *MemberBlockRepository) List(blockerSeq int) ([]model.MemberBlockState, error) {
	var rows []struct {
		Name       string         `db:"NAME"`
		PhotoURL   sql.NullString `db:"PHOTO_URL"`
		Cohort     sql.NullString `db:"COHORT"`
		Department sql.NullString `db:"DEPARTMENT"`
		UserSeq    int            `db:"BLOCKED_USR_SEQ"`
		UpdatedAt  string         `db:"UPDATED_AT"`
	}
	if err := r.DB.Select(&rows, `
		SELECT b.BLOCKED_USR_SEQ,
 DATE_FORMAT(b.UPDATED_AT, '%Y-%m-%dT%H:%i:%sZ') AS UPDATED_AT,
 CASE WHEN m.USR_STATUS IN ('BBB','CCC','ZZZ') THEN COALESCE(NULLIF(m.USR_NAME,''),'이름 없음') ELSE '탈퇴한 회원' END AS NAME,
 CASE WHEN m.USR_STATUS IN ('BBB','CCC','ZZZ') THEN m.USR_PHOTO ELSE NULL END AS PHOTO_URL,
 CASE WHEN m.USR_STATUS IN ('BBB','CCC','ZZZ') THEN v.COHORT ELSE NULL END AS COHORT,
 CASE WHEN m.USR_STATUS IN ('BBB','CCC','ZZZ') THEN v.DEPARTMENT ELSE NULL END AS DEPARTMENT
 FROM ALUMNI_MEMBER_BLOCK b
 LEFT JOIN WEO_MEMBER m ON m.USR_SEQ = b.BLOCKED_USR_SEQ
 LEFT JOIN ALUMNI_VERIFICATION v ON v.USR_SEQ = b.BLOCKED_USR_SEQ
 WHERE b.BLOCKER_USR_SEQ = ?
 ORDER BY b.UPDATED_AT DESC, b.BLOCKED_USR_SEQ DESC
	`, blockerSeq); err != nil {
		return nil, err
	}
	states := make([]model.MemberBlockState, 0, len(rows))
	for _, row := range rows {
		updatedAt := row.UpdatedAt
		states = append(states, model.MemberBlockState{
			Name:        row.Name,
			PhotoURL:    blockNullableString(row.PhotoURL),
			Cohort:      row.Cohort.String,
			Department:  row.Department.String,
			UserSeq:     row.UserSeq,
			BlockedByMe: true,
			UpdatedAt:   &updatedAt,
		})
	}
	return states, nil
}

func (r *MemberBlockRepository) Get(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	var updatedAt string
	if err := r.DB.Get(&updatedAt, `
		SELECT DATE_FORMAT(UPDATED_AT, '%Y-%m-%dT%H:%i:%sZ')
		FROM ALUMNI_MEMBER_BLOCK
		WHERE BLOCKER_USR_SEQ = ? AND BLOCKED_USR_SEQ = ?
		LIMIT 1
	`, blockerSeq, blockedSeq); err != nil {
		if err == sql.ErrNoRows {
			return &model.MemberBlockState{UserSeq: blockedSeq, BlockedByMe: false, UpdatedAt: nil}, nil
		}
		return nil, err
	}
	return &model.MemberBlockState{UserSeq: blockedSeq, BlockedByMe: true, UpdatedAt: &updatedAt}, nil
}

func (r *MemberBlockRepository) Block(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var status string
	if err := tx.Get(&status, `
		SELECT v.STATUS
		FROM ALUMNI_VERIFICATION v
		JOIN WEO_MEMBER m ON m.USR_SEQ = v.USR_SEQ
		WHERE v.USR_SEQ = ?
		  AND m.USR_STATUS IN ('BBB','CCC','ZZZ')
		FOR UPDATE
	`, blockedSeq); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMemberBlockTargetNotApproved
		}
		return nil, err
	}
	if status != "approved" {
		return nil, ErrMemberBlockTargetNotApproved
	}

	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_MEMBER_BLOCK (
			BLOCKER_USR_SEQ, BLOCKED_USR_SEQ, CREATED_AT, UPDATED_AT
		) VALUES (?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE UPDATED_AT = UPDATED_AT
	`, blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	var updatedAt string
	if err := tx.Get(&updatedAt, `
		SELECT DATE_FORMAT(UPDATED_AT, '%Y-%m-%dT%H:%i:%sZ')
		FROM ALUMNI_MEMBER_BLOCK
		WHERE BLOCKER_USR_SEQ = ? AND BLOCKED_USR_SEQ = ?
		LIMIT 1
	`, blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.MemberBlockState{UserSeq: blockedSeq, BlockedByMe: true, UpdatedAt: &updatedAt}, nil
}

func (r *MemberBlockRepository) Unblock(blockerSeq, blockedSeq int) (*model.MemberBlockState, error) {
	if _, err := r.DB.Exec(`
		DELETE FROM ALUMNI_MEMBER_BLOCK
		WHERE BLOCKER_USR_SEQ = ? AND BLOCKED_USR_SEQ = ?
	`, blockerSeq, blockedSeq); err != nil {
		return nil, err
	}
	return &model.MemberBlockState{UserSeq: blockedSeq, BlockedByMe: false, UpdatedAt: nil}, nil
}

func (r *MemberBlockRepository) DeleteExpiredSuppressedMessages(limit int) (int64, error) {
	result, err := r.DB.Exec(`
		DELETE FROM ALUMNI_MESSAGE
		WHERE PURGE_AT IS NOT NULL
			AND PURGE_AT <= UTC_TIMESTAMP()
			AND AM_VISIBLE_RECVR = 'N'
			AND AM_SUPPRESSION_REASON = 'recipient_blocked'
		ORDER BY PURGE_AT ASC, AM_SEQ ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func blockNullableString(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}
