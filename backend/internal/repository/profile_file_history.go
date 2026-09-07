// profile_file_history.go — Preserve managed image references before replacement.
package repository

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

// column is code-owned, never an HTTP field or arbitrary SQL identifier.
func rememberProfileFile(tx *sqlx.Tx, user int, column string) error {
	if column != "USR_PHOTO" && column != "USR_BIZ_CARD" {
		return sql.ErrNoRows
	}
	_, err := tx.Exec(`INSERT IGNORE INTO ALUMNI_PROFILE_FILE_HISTORY (USR_SEQ,URL_HASH,URL_PATH,RECORDED_AT)
 SELECT USR_SEQ,SHA2(`+column+`,256),`+column+`,UTC_TIMESTAMP() FROM WEO_MEMBER
 WHERE USR_SEQ=? AND (`+column+` LIKE '/uploads/%' OR `+column+` LIKE '/files/%' OR `+column+` LIKE '/upload/%' OR `+column+` LIKE '/old/upload/%')`, user)
	return err
}

func (r *ProfileRepository) replaceProfileFile(user int, column, url string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err = tx.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=? FOR UPDATE`, user); err != nil {
		return err
	}
	if status == "AAA" {
		return sql.ErrNoRows
	}
	if err = rememberProfileFile(tx, user, column); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE WEO_MEMBER SET "+column+"=? WHERE USR_SEQ=?", url, user); err != nil {
		return err
	}
	return tx.Commit()
}
