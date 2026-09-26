// admin_operator.go — Admin role holder rows for the root-only operator management screen
package model

import "time"

// AdminOperator is one ALUMNI_ADMIN_ROLE holder with the member fields the
// operator management screen shows.
type AdminOperator struct {
	USRSeq        int       `db:"USR_SEQ" json:"usrSeq"`
	USRID         string    `db:"USR_ID" json:"usrId"`
	USRName       string    `db:"USR_NAME" json:"usrName"`
	AdminRole     AdminRole `db:"ADMIN_ROLE" json:"adminRole"`
	CreatedAt     time.Time `db:"CREATED_AT" json:"createdAt"`
	UpdatedAt     time.Time `db:"UPDATED_AT" json:"updatedAt"`
	UpdatedByName string    `db:"UPDATED_BY_NAME" json:"updatedByName"`
}
