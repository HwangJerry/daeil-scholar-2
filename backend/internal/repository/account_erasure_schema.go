// account_erasure_schema.go — Explicit deletion targets with transactional schema checks.
package repository

import (
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type erasureSchema map[string]map[string]bool

func readErasureSchema(tx *sqlx.Tx) (erasureSchema, error) {
	var rows []struct {
		Table  string `db:"TABLE_NAME"`
		Column string `db:"COLUMN_NAME"`
	}
	err := tx.Select(&rows, `SELECT TABLE_NAME,COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE()`)
	s := erasureSchema{}
	for _, r := range rows {
		if s[r.Table] == nil {
			s[r.Table] = map[string]bool{}
		}
		s[r.Table][r.Column] = true
	}
	return s, err
}
func (s erasureSchema) has(t, c string) bool { return s[t][c] }
func (s erasureSchema) erase(tx *sqlx.Tx, table, where string, args ...interface{}) error {
	if s[table] == nil {
		return nil
	}
	// Tables and predicates are code-owned constants, never HTTP input.
	var engine string
	if err := tx.Get(&engine, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table); err != nil {
		return err
	}
	if engine != "InnoDB" {
		return &model.ErasureBlocked{Code: "NON_TRANSACTIONAL_TABLE_" + table}
	}
	_, err := tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE %s", table, where), args...)
	return err
}

var erasureMemberTables = []string{
	"ALUMNI_PUSH_OUTBOX", "ALUMNI_MOBILE_DEVICE_TOKEN", "ALUMNI_PUSH_DEVICE", "ALUMNI_PUSH_PREFERENCE",
	"ALUMNI_MOBILE_REFRESH_TOKEN", "ALUMNI_NOTIFICATION", "ALUMNI_PASSWORD_RESET", "USER_SESSION",
	"WEO_MEMBER_LOG", "WEO_MEMBER_PUSH", "WEO_MEMBER_OUT", "WEO_PUSH_NOTI", "WEO_SMS", "WEO_APP_PUSH",
	"ALUMNI_USER_TAG", "ALUMNI_ADMIN_ROLE", "ALUMNI_VERIFICATION", "FUNDAMENTAL_MEMBER",
	"WEO_BOARDLIKE", "WEO_BOARDCOMAND", "WEO_AD_COMMENT", "WEO_AD_LIKE", "WEO_AD_LOG", "WEO_BANNER_AD_LOG",
}

func eraseAccountReferences(tx *sqlx.Tx, s erasureSchema, user int, email string) error {
	if s["ALUMNI_MESSAGE"] != nil {
		for _, q := range []struct{ table, where string }{
			{"ALUMNI_PUSH_OUTBOX", "EVENT_TYPE='message' AND EVENT_ID IN (SELECT AM_SEQ FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?)"},
			{"ALUMNI_NOTIFICATION", "AN_TYPE='message' AND AN_REF_SEQ IN (SELECT AM_SEQ FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?)"},
		} {
			if err := s.erase(tx, q.table, q.where, user, user); err != nil {
				return err
			}
		}
	}
	for _, table := range erasureMemberTables {
		if err := s.erase(tx, table, "USR_SEQ=?", user); err != nil {
			return err
		}
	}
	for _, q := range []struct {
		table, where string
		args         []interface{}
	}{
		{"ALUMNI_MESSAGE_REPORT", "REPORTER_SEQ=? OR REPORTED_SEQ=?", []interface{}{user, user}},
		{"ALUMNI_MEMBER_BLOCK", "BLOCKER_USR_SEQ=? OR BLOCKED_USR_SEQ=?", []interface{}{user, user}},
		{"ALUMNI_MESSAGE", "AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?", []interface{}{user, user}},
		{"ALUMNI_MOBILE_APP_EVENT", "USER_ID=?", []interface{}{user}},
		{"WEO_VISIT_DAILY", "VD_USR_SEQ=?", []interface{}{user}},
		{"AUTH_EMAIL_VERIFICATION", "NORMALIZED_EMAIL=? AND ?<>''", []interface{}{email, email}},
	} {
		if err := s.erase(tx, q.table, q.where, q.args...); err != nil {
			return err
		}
	}
	if s["AUTH_IDENTITY"] != nil {
		if err := s.erase(tx, "AUTH_EMAIL_VERIFICATION", "NORMALIZED_EMAIL IN (SELECT NORMALIZED_EMAIL FROM AUTH_IDENTITY WHERE ACCOUNT_ID=?)", user); err != nil {
			return err
		}
		if s["AUTH_SIGNUP_CONTINUATION"] != nil {
			subjectMatch := "EXISTS (SELECT 1 FROM AUTH_IDENTITY i WHERE i.ACCOUNT_ID=? AND i.PROVIDER=AUTH_SIGNUP_CONTINUATION.PROVIDER AND i.SUBJECT_KEY=AUTH_SIGNUP_CONTINUATION.SUBJECT_KEY)"
			if err := s.erase(tx, "AUTH_PROVIDER_CREDENTIAL", "CONTINUATION_TOKEN_HASH IN (SELECT TOKEN_HASH FROM AUTH_SIGNUP_CONTINUATION WHERE "+subjectMatch+")", user); err != nil {
				return err
			}
			if err := s.erase(tx, "AUTH_SIGNUP_CONTINUATION", subjectMatch, user); err != nil {
				return err
			}
		}

		for _, t := range []string{"AUTH_PROVIDER_REVOKE_OUTBOX", "AUTH_SESSION_FAMILY", "AUTH_PASSWORD_CREDENTIAL", "AUTH_PROVIDER_CREDENTIAL"} {
			if err := s.erase(tx, t, "IDENTITY_ID IN (SELECT IDENTITY_ID FROM AUTH_IDENTITY WHERE ACCOUNT_ID=?)", user); err != nil {
				return err
			}
		}
	}
	for _, t := range []string{"AUTH_PHONE_CLAIM", "AUTH_CONSENT", "AUTH_IDENTITY", "AUTH_ACCOUNT_STATE"} {
		if err := s.erase(tx, t, "ACCOUNT_ID=?", user); err != nil {
			return err
		}
	}
	for _, t := range []string{"WEO_MEMBER_SOCIAL", "ALUMNI_SOCIAL_CREDENTIAL", "ALUMNI_UPLOAD_OWNER"} {
		if err := s.erase(tx, t, "USR_SEQ=?", user); err != nil {
			return err
		}
	}
	return nil
}
