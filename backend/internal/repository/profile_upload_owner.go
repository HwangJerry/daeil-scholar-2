// profile_upload_owner.go — Attach profile uploads under the account lifecycle lock.
package repository

import "database/sql"

func (r *ProfileRepository) AssignProfileUpload(user, fileID int, url string, businessCard bool) error {
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
	if _, err = tx.Exec(`INSERT INTO ALUMNI_UPLOAD_OWNER (F_SEQ,USR_SEQ,URL_PATH) VALUES (?,?,?)`, fileID, user, url); err != nil {
		return err
	}
	column := "USR_PHOTO"
	if businessCard {
		column = "USR_BIZ_CARD"
	}
	if _, err = tx.Exec("UPDATE WEO_MEMBER SET "+column+"=? WHERE USR_SEQ=?", url, user); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FileRepository) DeleteUnassignedUpload(id int) error {
	_, err := r.DB.Exec(`DELETE FROM WEO_FILES WHERE F_SEQ=? AND F_JOIN_SEQ=0 AND NOT EXISTS (SELECT 1 FROM ALUMNI_UPLOAD_OWNER WHERE F_SEQ=?)`, id, id)
	return err
}
